#!/usr/bin/env bash
# 一键编译并部署 mp-server 到服务器：
#   - 本地交叉编译 linux/amd64 静态二进制（纯 Go，无需服务器装 Go）
#   - scp 到 $REMOTE_DIR，首次部署生成 config.yaml（含 admin_token）
#   - 安装/启用 systemd 服务 mp-helper（开机自启）
#   - 安装 nginx 站点配置，nginx -t 通过后 reload
#   - healthz 自检
#
# 用法：
#   cp deploy/.env.example deploy/.env   # 填好真实值（.env 已被忽略）
#   ./deploy/deploy.sh
#
# 依赖：本地已装 go；本机对服务器有 SSH 免密（key）权限。
set -euo pipefail

# 切到仓库根目录
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ENV_FILE="deploy/.env"
if [ ! -f "$ENV_FILE" ]; then
  echo "缺少 $ENV_FILE，请先复制 deploy/.env.example 为 deploy/.env 并填写。" >&2
  exit 1
fi
# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a

: "${SSH_HOST:?}"; : "${SSH_PORT:?}"; : "${SSH_USER:?}"
: "${DOMAIN:?}"; : "${REMOTE_DIR:?}"; : "${BIND_ADDR:?}"; : "${ADMIN_TOKEN:?}"
: "${NGINX_CONTAINER:?}"; : "${NGINX_CONFD:?}"

SSH_OPTS=(-p "$SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=20)
SCP_OPTS=(-P "$SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=20)
REMOTE="${SSH_USER}@${SSH_HOST}"

echo ">> [1/6] 交叉编译 linux/amd64"
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/mp-server ./cmd/mp-server
ls -lh dist/mp-server

echo ">> [2/6] 渲染 config.yaml（本地，含密钥，dist 已被忽略）"
sed -e "s|__BIND__|${BIND_ADDR}|" \
    -e "s|__ADMIN__|${ADMIN_TOKEN}|" \
    -e "s|__DBPATH__|${REMOTE_DIR}/mp-helper.db|" \
    deploy/config.yaml.tmpl > dist/config.yaml

echo ">> [3/6] 准备远端目录"
ssh "${SSH_OPTS[@]}" "$REMOTE" "mkdir -p '${REMOTE_DIR}'"

echo ">> [4/6] 上传二进制 / 配置 / 单元文件 / nginx 站点"
scp "${SCP_OPTS[@]}" dist/mp-server                              "${REMOTE}:${REMOTE_DIR}/mp-server.new"
scp "${SCP_OPTS[@]}" dist/config.yaml                            "${REMOTE}:${REMOTE_DIR}/config.yaml.new"
scp "${SCP_OPTS[@]}" deploy/mp-helper.service                    "${REMOTE}:/etc/systemd/system/mp-helper.service"
scp "${SCP_OPTS[@]}" deploy/nginx-mp-helper.aworld.ltd.conf      "${REMOTE}:${NGINX_CONFD}/${DOMAIN}.conf"

echo ">> [5/6] 远端：配置/二进制切换、systemd、nginx reload、健康检查"
ssh "${SSH_OPTS[@]}" "$REMOTE" "bash -s '${REMOTE_DIR}' '${DOMAIN}' '${NGINX_CONTAINER}' '${BIND_ADDR}'" <<'EOSSH'
set -euo pipefail
REMOTE_DIR="$1"; DOMAIN="$2"; NGINX_CONTAINER="$3"; BIND_ADDR="$4"
cd "$REMOTE_DIR"

# 首次部署才落 config（避免覆盖已有配置/密钥）
if [ -f config.yaml ]; then
  rm -f config.yaml.new
  echo "   config.yaml 已存在，保留不覆盖"
else
  mv config.yaml.new config.yaml
  chmod 600 config.yaml
  echo "   config.yaml 已创建"
fi

# 原子替换二进制
systemctl daemon-reload
systemctl stop mp-helper 2>/dev/null || true
mv -f mp-server.new mp-server
chmod +x mp-server
systemctl enable mp-helper >/dev/null 2>&1 || true
systemctl restart mp-helper
sleep 1
if ! systemctl is-active --quiet mp-helper; then
  echo "   mp-helper 未启动，最近日志：" >&2
  journalctl -u mp-helper --no-pager -n 30 >&2
  exit 1
fi
echo "   systemd: mp-helper active"

# nginx 校验通过才 reload（失败不影响既有站点）
if docker exec "$NGINX_CONTAINER" nginx -t; then
  docker exec "$NGINX_CONTAINER" nginx -s reload
  echo "   nginx reloaded"
else
  echo "   nginx -t 失败，未 reload" >&2
  exit 1
fi

# 健康检查（宿主直连应用）
sleep 1
if curl -fsS "http://${BIND_ADDR}/healthz" >/dev/null; then
  echo "   healthz: OK (http://${BIND_ADDR}/healthz)"
else
  echo "   healthz 失败" >&2
  journalctl -u mp-helper --no-pager -n 30 >&2
  exit 1
fi
EOSSH

echo ">> [6/6] 完成。外部访问：https://${DOMAIN}/healthz （需 DNS A 记录指向 ${SSH_HOST}）"

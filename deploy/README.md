# 部署 mp-helper（一键）

把 `mp-server` 部署到服务器，注册为 systemd 服务（开机自启），并通过宿主上的 docker nginx 反代到域名。

## 架构

```
客户端 ──HTTPS──> nginx(docker, base-nginx, 443)
                     └─ proxy_pass http://172.17.0.1:8088
                           └─ mp-server (systemd, 监听 172.17.0.1:8088)
                                 └─ SQLite: /opt/mp-helper/mp-helper.db
```

- mp-server 监听 `172.17.0.1:8088`（docker0 网关），**不暴露公网**，仅 nginx/宿主可达。
- nginx 复用现有 `*.aworld.ltd` 通配证书，无需为子域单独签证书。
- 鉴权由应用负责：数据面用业务 API Key、管理面用 admin token；nginx 仅透传。

## 一次性准备

```bash
cp deploy/.env.example deploy/.env
# 编辑 deploy/.env，至少填 ADMIN_TOKEN；deploy/.env 已被 .gitignore 忽略
```

本机需：已装 `go`，且对服务器有 SSH 免密（key）权限。

## 部署 / 更新

```bash
./deploy/deploy.sh
```

脚本会：交叉编译 linux/amd64 → 上传 → 首次生成 `config.yaml` → 启用并重启 `mp-helper` 服务 → `nginx -t` 通过后 reload → healthz 自检。
重复执行即“更新”：仅替换二进制并重启，**不覆盖**已有 `config.yaml` 与数据库。

## 发布 mp-cli 多平台下载

把 `mp-cli` 编译为 linux/amd64、windows/amd64、darwin/arm64，并发布到服务器，经 nginx 静态目录对外提供下载（供安装 skill 时按平台直接下载，无需 Go）：

```bash
./deploy/release-cli.sh
```

产出下载地址：
- `https://mp-helper.aworld.ltd/download/mp-cli-linux-amd64`
- `https://mp-helper.aworld.ltd/download/mp-cli-darwin-arm64`
- `https://mp-helper.aworld.ltd/download/mp-cli-windows-amd64.exe`
- `https://mp-helper.aworld.ltd/download/sha256sums.txt`

nginx 站点配置里的 `location /download/` 指向 `/usr/share/nginx/html/mp-helper/download/`（宿主 `/opt/docker/nginx/html/mp-helper/download/`）。

## DNS

外部访问需把 `mp-helper.aworld.ltd` 的 A 记录指向服务器公网 IP（`8.138.43.109`）。

## 服务器常用运维

```bash
systemctl status mp-helper          # 状态
journalctl -u mp-helper -f          # 实时日志
systemctl restart mp-helper         # 重启
cat /opt/mp-helper/config.yaml      # 配置（含 admin_token）
```

## 配置公众号与签发业务 Key（管理面，需 admin token）

```bash
ADMIN="Authorization: Bearer <admin_token>"
BASE="https://mp-helper.aworld.ltd"

curl -s -X POST "$BASE/admin/accounts" -H "$ADMIN" -H 'Content-Type: application/json' \
  -d '{"name":"gz","appid":"wx...","app_secret":"..."}'

curl -s -X POST "$BASE/admin/accounts/1/keys" -H "$ADMIN" -H 'Content-Type: application/json' \
  -d '{"label":"agent-A"}'   # 返回的明文 key 仅此一次
```

把业务 key 配到使用方的 `MP_HELPER_API_KEY`，`MP_HELPER_API_URL=https://mp-helper.aworld.ltd`。

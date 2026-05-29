#!/usr/bin/env bash
# 编译 mp-cli 多平台二进制并发布到服务器，供下载安装：
#   https://<DOMAIN>/download/mp-cli-<os>-<arch>[.exe]
#   https://<DOMAIN>/download/sha256sums.txt
#
# 平台：linux/amd64、windows/amd64、darwin/arm64
# 用法：./deploy/release-cli.sh    （读取 deploy/.env）
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ENV_FILE="deploy/.env"
[ -f "$ENV_FILE" ] || { echo "缺少 $ENV_FILE" >&2; exit 1; }
# shellcheck disable=SC1090
set -a; source "$ENV_FILE"; set +a
: "${SSH_HOST:?}"; : "${SSH_PORT:?}"; : "${SSH_USER:?}"; : "${DOMAIN:?}"
: "${NGINX_HTML:?}"

SSH_OPTS=(-p "$SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=20)
SCP_OPTS=(-P "$SSH_PORT" -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ConnectTimeout=20)
REMOTE="${SSH_USER}@${SSH_HOST}"
DL_DIR="${NGINX_HTML}/mp-helper/download"

OUT="dist/release"
rm -rf "$OUT"; mkdir -p "$OUT"

build() { # GOOS GOARCH outfile
  echo ">> build $1/$2 -> $3"
  CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags="-s -w" -o "${OUT}/$3" ./cmd/mp-cli
}
build linux   amd64 mp-cli-linux-amd64
build windows amd64 mp-cli-windows-amd64.exe
build darwin  arm64 mp-cli-darwin-arm64

echo ">> sha256"
( cd "$OUT" && shasum -a 256 mp-cli-* > sha256sums.txt && cat sha256sums.txt )

echo ">> upload to ${REMOTE}:${DL_DIR}"
ssh "${SSH_OPTS[@]}" "$REMOTE" "mkdir -p '${DL_DIR}'"
scp "${SCP_OPTS[@]}" "${OUT}"/* "${REMOTE}:${DL_DIR}/"

echo ">> 完成。下载地址："
echo "   https://${DOMAIN}/download/mp-cli-linux-amd64"
echo "   https://${DOMAIN}/download/mp-cli-windows-amd64.exe"
echo "   https://${DOMAIN}/download/mp-cli-darwin-arm64"
echo "   https://${DOMAIN}/download/sha256sums.txt"

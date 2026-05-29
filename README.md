# mp-helper

微信公众号助手：Web 服务（素材上传 / 创建草稿 / 管理接口）+ CLI 工具 + 通用 agent skill。底层用 [silenceper/wechat](https://github.com/silenceper/wechat) v2。

- 设计文档：`docs/superpowers/specs/2026-05-29-mp-helper-design.md`
- 安装 skill（面向 agent）：`INSTALL.md`

## 构建

需要 Go 1.25+（go.mod 声明 `go 1.25`；若本机是 Go 1.21+，在有网络时 Go 会自动下载所需 toolchain）。

```bash
make build      # 产出 bin/mp-server 与 bin/mp-cli
```

## 部署 mp-server

1. 准备配置（参考 `configs/config.example.yaml`）：
   ```yaml
   listen: ":8080"
   db_path: "/var/lib/mp-helper/mp-helper.db"
   admin_token: "一个足够强的随机串"
   cache:
     type: "memory"
   wechat_timeout_seconds: 30
   ```
   `admin_token` 也可用环境变量 `MP_HELPER_ADMIN_TOKEN` 覆盖。

2. 启动：
   ```bash
   ./bin/mp-server --config config.yaml
   ```
   建议置于 HTTPS 反向代理之后；`config.yaml` 与 SQLite 文件权限设为 600。

## 管理公众号与 API Key（带 admin token）

```bash
ADMIN="Authorization: Bearer <admin_token>"
BASE="http://127.0.0.1:8080"

# 新增公众号
curl -s -X POST "$BASE/admin/accounts" -H "$ADMIN" -H 'Content-Type: application/json' \
  -d '{"name":"gz","appid":"wx...","app_secret":"..."}'

# 为账号 1 生成业务 API Key（明文仅此一次返回）
curl -s -X POST "$BASE/admin/accounts/1/keys" -H "$ADMIN" -H 'Content-Type: application/json' \
  -d '{"label":"agent-A"}'

# 列出 / 吊销 key
curl -s "$BASE/admin/keys" -H "$ADMIN"
curl -s -X DELETE "$BASE/admin/keys/1" -H "$ADMIN"
```

把生成的业务 API Key 配到使用方的 `MP_HELPER_API_KEY`。

## 数据面接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` | 健康检查 |
| POST | `/api/v1/materials` | multipart 上传图片，返回 `{media_id,url}` |
| POST | `/api/v1/drafts` | JSON 文章列表，返回 `{media_id}` |

数据面需请求头 `Authorization: Bearer <api_key>`，key 唯一对应一个公众号。

## 安全与已知限制

- `app_secret` 在 SQLite 中明文存储（调用微信必需）；依赖文件系统权限保护。API Key 仅存 sha256 哈希。
- token 缓存为进程内存：多 worker 部署时各自缓存 access_token（功能正常，可能多刷几次）。需要共享请后续接 Redis。
- 微信调用层（素材/草稿）只能用真实凭证集成测试覆盖；CI 默认跳过。

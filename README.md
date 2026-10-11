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
   archive:            # 每日统计存档（可省略，默认如下）
     enabled: true
     time: "09:30"     # 北京时间
     max_calls: 200    # 每个公众号每次最多调用微信的次数
   ```
   `admin_token` 也可用环境变量 `MP_HELPER_ADMIN_TOKEN` 覆盖。

   每日统计存档：每天按 `archive.time` 把各公众号缺失或未定稿的统计（单篇明细、用户增减、累计、阅读汇总）补进 SQLite，
   首次启用时从 2025-11-01 起回补，每次受 `max_calls` 限制，几天内补齐；遇到 48001（无权限）或 45009（当日次数用完）即停止该账号。
   分析接口优先读存档，微信下线或调整接口时历史数据仍在。

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

# 立即执行一次统计存档（后台运行），并查看最近一次结果
curl -s -X POST "$BASE/admin/archive/run" -H "$ADMIN"
curl -s "$BASE/admin/archive" -H "$ADMIN"
```

把生成的业务 API Key 配到使用方的 `MP_HELPER_API_KEY`。

## 数据面接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz` | 健康检查 |
| POST | `/api/v1/materials` | multipart 上传图片，返回 `{media_id,url}` |
| POST | `/api/v1/drafts` | JSON 文章列表，返回 `{media_id}` |
| PUT | `/api/v1/drafts` | 更新已有草稿：`{media_id, index, articles}`，`articles` 为单篇完整文章对象（九个字段必须写全）；只写一次，不改为新建、不自动重试 |
| POST | `/api/v1/body-images` | multipart `file`，jpg/png 小于 1 MB，原字节转发微信 `media/uploadimg`，返回 `{url}`（正文图片，不进永久素材库） |
| GET | `/api/v1/drafts?offset=&count=&no_content=` | 草稿列表（只读，微信原始返回；count 1–20，no_content 缺省 1） |
| GET | `/api/v1/drafts/:media_id` | 读取一篇草稿（保存后回读核对） |
| GET | `/api/v1/published?offset=&count=&no_content=` | 已发布图文列表（只读；「群发」的图文不在其中） |
| GET | `/api/v1/published/:article_id` | 读取一篇已发布图文（含正文） |
| GET | `/api/v1/datacube/:api?begin=YYYY-MM-DD&end=YYYY-MM-DD` | 数据统计透传，只放行 `getarticletotaldetail`（单篇逐日明细，含群发）、`getarticleread`、`getarticleshare`、`getbizsummary`、`getusersummary`、`getusercumulate` |
| GET | `/api/v1/analytics/articles?begin=&end=&series=` | 按发表日期汇总单篇表现：当天 / 7 天 / 最新的阅读、来源、分享、收藏，推送打开率、推算送达人数、推送时刻（人工补录优先，其次取当天的发布记录）；附按分类、按推送小时的分组汇总 |
| GET | `/api/v1/analytics/daily?begin=&end=` | 每日新增 / 取关 / 累计、取关率、阅读汇总、推送篇数、日类型（推送日 / 次日 / 不发日）与近 7 天发文节奏（每天 / 隔天 / 偶尔 / 不发）；附按日类型、按节奏的平均与节奏区间 |
| PUT | `/api/v1/articles/:msgid/meta` | 补录文章推送时刻（`published_at`，北京时间 `YYYY-MM-DD HH:MM`）、分类、备注；省略的字段保留原值 |
| GET | `/api/v1/articles/meta` | 列出补录信息 |

数据面需请求头 `Authorization: Bearer <api_key>`，key 唯一对应一个公众号。`analytics` 两个接口的日期须满足 begin ≥ 2025-11-01、end ≤ 昨天（北京时间）、跨度 ≤ 31 天；个别日期取数失败记入 `warnings`，全部失败才返回 502。统计数据按天缓存在 SQLite（与每日存档共用）：单篇明细过了 30 天统计期、其余统计早于前天即视为定稿，永久保存；未定稿的缓存 1 小时。除 `analytics` 外，只读接口原样透传微信返回，微信报错时返回 502 并带 `wechat_errcode`（如 48001 未授权、47009 接口已下线）。

## 安全与已知限制

- `app_secret` 在 SQLite 中明文存储（调用微信必需）；依赖文件系统权限保护。API Key 仅存 sha256 哈希。
- token 缓存为进程内存：多 worker 部署时各自缓存 access_token（功能正常，可能多刷几次）。需要共享请后续接 Redis。
- 微信调用层（素材/草稿）只能用真实凭证集成测试覆盖；CI 默认跳过。

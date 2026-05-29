# mp-helper 设计文档

- 日期：2026-05-29
- 状态：待评审
- 作者：Claude Code + iazure

## 1. 目标与背景

为 AI agent 提供一套「微信公众号素材上传 + 草稿箱发布」的能力，让 agent 通过 skill 调用一个 CLI 工具，CLI 请求一个运行在服务器上的 Web 服务，Web 服务用 Go 版微信 SDK（[silenceper/wechat](https://github.com/silenceper/wechat) v2）实现对微信公众平台的调用。

项目交付三类可用产物：

1. **Web 服务**（`mp-server`）：跑在服务器上，对外暴露 HTTP 接口（素材上传、创建草稿、管理接口），用 SQLite 持久化公众号与 API Key 数据。
2. **CLI 工具**（`mp-cli`）：供 skill 调用，把命令翻译成对 Web 服务的 HTTP 请求。
3. **Skill + 安装文档**：通用 agent skill（`SKILL.md`，不绑定特定 agent 工具）+ 面向 agent 的机器可执行安装文档（`INSTALL.md`）。

### 非目标（Out of Scope）

- 不做语音/视频素材上传（首版只做图片永久素材，YAGNI）。
- 不做「群发/发布」（`freepublish`）——只到草稿箱为止。
- 不做公众号消息收发、菜单、客服等其它能力。
- 不做 Web 管理界面（管理只提供 HTTP 接口 + curl）。
- 不做 app_secret 静态加密（库内明文存储，文档注明取舍）。

## 2. 技术选型（已确认）

| 维度 | 选择 | 说明 |
|---|---|---|
| 语言 | Go | Web 服务与 CLI 均用 Go |
| 微信 SDK | silenceper/wechat v2 | `officialaccount/material`、`officialaccount/draft` |
| Web 框架 | Gin | 路由/中间件/绑定齐全 |
| 仓库布局 | 单 Go module + `cmd/` 双二进制 | 共享 `internal/types` |
| CLI 框架 | cobra | Go CLI 事实标准 |
| 鉴权 | 静态 API Key（Bearer） | 数据面用 key；管理面用 admin token |
| 多租户 | 1 个 API Key 绑 1 个公众号 | key 既鉴权又选账号 |
| 数据存储 | SQLite | 账号与 API Key 数据 |
| SQLite 驱动 | `modernc.org/sqlite`（纯 Go） | 无需 CGO，跨平台 `go build` 无障碍 |
| 数据访问 | `database/sql` + 原生 SQL + 内嵌 `schema.sql` | 不引 ORM |
| 运行配置 | YAML（`gopkg.in/yaml.v3`） | 端口、DB 路径、admin token、Redis 开关 |
| token 缓存 | 内存（默认），可选 Redis | 复用 silenceper `cache` |
| Skill 二进制分发 | 源码 `go build` | 需 Go 1.22+ 工具链 |
| Skill 形态 | 通用 agent skill（`SKILL.md`） | 不绑定特定 agent 工具 |

## 3. 架构与数据流

```
AI agent
  └─(触发 skill)→ SKILL.md 指示调用 CLI
        └→ mp-cli (Go)            读取 env: MP_HELPER_API_URL / MP_HELPER_API_KEY
              └─(HTTPS, Bearer key)→ mp-server (Gin)
                    ├─ auth 中间件: key 哈希 → 查 SQLite → 得到 account(appid/secret)
                    ├─ wechat.manager: 按 appid 取/建 OfficialAccount 实例(共享 token cache)
                    └─ silenceper/wechat → api.weixin.qq.com
```

管理面（人工/运维）：

```
运维 ──(HTTPS, admin token)→ mp-server /admin/* → SQLite (accounts / api_keys)
```

数据面与管理面跑在同一个 `mp-server` 进程里，用不同的鉴权（业务 API Key vs admin token）隔离。

## 4. 仓库结构

```
mp-helper/
├── go.mod / go.sum                  # module github.com/iazure/mp-helper
├── Makefile                         # build/test/lint
├── README.md                        # 面向人类：部署 Web 服务
├── INSTALL.md                       # 面向 agent：安装 skill（机器可执行步骤）
├── cmd/
│   ├── mp-server/main.go            # 加载 YAML 配置、打开 DB、装配 Gin、启动
│   └── mp-cli/main.go               # cobra 根命令
├── internal/
│   ├── api/
│   │   ├── server.go                # 路由装配、依赖注入
│   │   ├── auth.go                  # 数据面 key 鉴权中间件 + 管理面 admin token 中间件
│   │   ├── handlers.go              # material / draft / health handler
│   │   ├── admin.go                 # /admin/* handler（账号、key 的 CRUD）
│   │   └── errors.go                # 统一错误响应与微信错误透传
│   ├── wechat/
│   │   ├── manager.go               # 按 appid 缓存 OfficialAccount，共享 token cache
│   │   ├── material.go              # 封装 AddMaterial
│   │   └── draft.go                 # 封装 AddDraft
│   ├── store/
│   │   ├── store.go                 # Open(dbPath)、迁移、连接管理
│   │   ├── accounts.go              # 账号 CRUD
│   │   ├── keys.go                  # API Key CRUD + 按哈希解析账号
│   │   └── schema.sql               # 内嵌（embed.FS）
│   ├── config/
│   │   └── config.go                # YAML 加载 + env 覆盖
│   ├── types/
│   │   └── types.go                 # 请求/响应 DTO（server & cli 共用）
│   └── client/
│       └── client.go                # CLI 用的 HTTP 客户端
├── configs/
│   └── config.example.yaml
├── skill/
│   └── mp-helper/
│       └── SKILL.md
└── docs/superpowers/specs/2026-05-29-mp-helper-design.md
```

## 5. 数据模型（SQLite）

`schema.sql`（通过 `embed.FS` 内嵌，启动时执行幂等建表）：

```sql
CREATE TABLE IF NOT EXISTS accounts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,           -- 账号别名，便于人识别
    appid       TEXT NOT NULL,
    app_secret  TEXT NOT NULL,                  -- 明文存储（调用微信需要）
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_keys (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL UNIQUE,           -- sha256(明文 key) hex
    prefix      TEXT NOT NULL,                  -- 明文前 8 位，用于列表展示
    label       TEXT,                           -- 备注
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at  DATETIME                        -- 非空表示已吊销
);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_account ON api_keys(account_id);
```

**映射关系**：一个 `api_key` 绑定唯一一个 `account`（`account_id` 外键）；一个 `account` 可挂多把 key（便于轮换/分发）。请求只带 key 即可确定公众号。

**安全约定**：

- API Key 明文形如 `mpk_<32 hex>`，仅在创建接口响应里返回一次；库内只存 `sha256` 哈希，无法二次读取。
- `app_secret` 明文存储（调用微信必需）。文档明确告知风险，依赖文件系统权限与服务器安全；后续可选「主密钥加密」作为增强。

## 6. 运行配置（YAML）

`configs/config.example.yaml`：

```yaml
# 监听地址与端口
listen: ":8080"

# SQLite 文件路径
db_path: "./mp-helper.db"

# 管理接口的 admin token（也可用环境变量 MP_HELPER_ADMIN_TOKEN 覆盖）
admin_token: "change-me-admin-token"

# access_token 缓存：默认内存；可选 Redis
cache:
  type: "memory"          # memory | redis
  redis:
    addr: "127.0.0.1:6379"
    password: ""
    db: 0

# 请求微信的超时（秒）
wechat_timeout_seconds: 30
```

加载顺序：默认值 → YAML 文件（`--config` flag 或 `MP_HELPER_CONFIG` env 指定）→ 环境变量覆盖（至少 `MP_HELPER_ADMIN_TOKEN`、`MP_HELPER_LISTEN`、`MP_HELPER_DB_PATH`）。

## 7. Web 服务接口

### 7.1 数据面（业务接口，需 `Authorization: Bearer <api_key>`）

#### `GET /healthz`
无需鉴权。返回 `{"status":"ok"}`。供 `mp-cli health` 与负载均衡探活。

#### `POST /api/v1/materials`
上传永久素材（首版仅图片）。`multipart/form-data`：

| 字段 | 必填 | 说明 |
|---|---|---|
| `file` | 是 | 图片文件 |
| `type` | 否 | 素材类型，默认 `image`（首版仅支持 `image`/`thumb`） |

响应：
```json
{ "media_id": "xxx", "url": "https://mmbiz.qpic.cn/..." }
```
`media_id` 可作封面 `thumb_media_id`；`url` 可嵌进正文 `<img src>`。底层调用 `material.AddMaterial`。

#### `POST /api/v1/drafts`
创建草稿。`application/json`：
```json
{
  "articles": [
    {
      "title": "标题",
      "author": "作者",
      "digest": "摘要",
      "content": "<p>正文 HTML</p>",
      "content_source_url": "https://...",
      "thumb_media_id": "封面图 media_id",
      "show_cover_pic": 1,
      "need_open_comment": 0,
      "only_fans_can_comment": 0
    }
  ]
}
```
响应：
```json
{ "media_id": "草稿 media_id" }
```
底层调用 `draft.AddDraft`。`articles` 至少一篇；字段映射到 silenceper 的 `draft.Article`。

### 7.2 管理面（`/admin/*`，需 `Authorization: Bearer <admin_token>`）

| 方法 | 路径 | 作用 |
|---|---|---|
| `POST` | `/admin/accounts` | 创建公众号 `{name, appid, app_secret}` → `{id, name, appid}` |
| `GET` | `/admin/accounts` | 列出公众号（不返回 secret） |
| `GET` | `/admin/accounts/{id}` | 查看单个公众号（不返回 secret） |
| `PUT` | `/admin/accounts/{id}` | 更新 `{name?, appid?, app_secret?}` |
| `DELETE` | `/admin/accounts/{id}` | 删除公众号（级联删除其 key） |
| `POST` | `/admin/accounts/{id}/keys` | 为账号创建 API Key `{label?}` → `{id, key, prefix}`（`key` 明文仅此一次） |
| `GET` | `/admin/keys` | 列出 key（`prefix/label/account_id/created_at/revoked_at`，不含明文） |
| `DELETE` | `/admin/keys/{id}` | 吊销 key（置 `revoked_at`） |

管理操作只读写 SQLite，不直接触达微信。账号被更新/删除时，`wechat.manager` 失效对应 appid 的缓存实例。

### 7.3 统一错误响应

```json
{ "error": { "code": "wechat_error", "message": "...", "wechat_errcode": 40007 } }
```
- 鉴权失败 → 401 `unauthorized`。
- 参数错误 → 400 `bad_request`。
- 账号/key 不存在 → 404 `not_found`。
- 微信返回 errcode≠0 → 502 `wechat_error`，透传 `wechat_errcode` 与 errmsg。

## 8. 微信集成（silenceper/wechat）

`internal/wechat/manager.go`：

- 持有一个共享的 `cache.Cache`（默认 `cache.NewMemory()`，配置为 redis 时用 `cache.NewRedis`）。
- `Get(account) (*officialaccount.OfficialAccount, error)`：按 `appid` 在 `map[string]*officialaccount.OfficialAccount`（带 `sync.RWMutex`）里取，没有则用账号的 appid/secret + 共享 cache 构造并缓存。
- `Invalidate(appid)`：账号更新/删除时清除实例。
- access_token 的获取/刷新/缓存由 silenceper 的 `credential` + `cache` 自动完成，多账号按 appid 隔离（缓存键含 appid）。

`material.go` 封装 `AddMaterial(MediaTypeImage, file)`；`draft.go` 把请求 DTO 转成 `[]*draft.Article` 调 `AddDraft`。微信请求超时按 YAML `wechat_timeout_seconds` 配置（通过 SDK 的 HTTP client 设置）。

## 9. CLI 工具（mp-cli）

cobra 命令，配置全部走环境变量（key 已决定账号，**无 `--account`**）：

| 环境变量 | 说明 |
|---|---|
| `MP_HELPER_API_URL` | Web 服务地址，如 `https://mp.example.com` |
| `MP_HELPER_API_KEY` | 业务 API Key（Bearer） |

命令：

```
mp-cli health
    # GET /healthz，打印服务状态

mp-cli material upload <file> [--type image]
    # 上传图片素材，stdout 输出 {"media_id":"...","url":"..."}

mp-cli draft create \
    --title "标题" \
    --content-file article.html \
    --cover <media_id 或 图片文件路径> \
    [--author 作者] [--digest 摘要] [--source-url URL] \
    [--show-cover-pic] [--need-open-comment] [--only-fans-can-comment]
    # 若 --cover 传的是文件路径，先自动上传得到 media_id 再建草稿
    # stdout 输出 {"media_id":"草稿 media_id"}
```

输出统一为 JSON（便于 agent 解析），错误写 stderr 且退出码非 0。`--json`/`--quiet` 视需要预留。管理操作不在 CLI 内（走 HTTP 管理接口）。

## 10. Skill 与安装文档

### 10.1 `skill/mp-helper/SKILL.md`

通用 agent skill 格式（YAML frontmatter + 正文），措辞 agent 中立：

- frontmatter：`name: mp-helper`，`description:` 触发条件（如「需要把内容发布到微信公众号草稿箱、上传公众号图片素材时」）。
- 正文：前置条件（已配置 `MP_HELPER_API_URL`/`MP_HELPER_API_KEY`、`mp-cli` 在 PATH）、标准工作流、命令示例、常见错误处理。
- 标准工作流：
  1. `mp-cli health` 自检；
  2. 准备封面图与正文图 → `mp-cli material upload` 取 media_id / url；
  3. 把正文图 url 嵌进 HTML 正文；
  4. `mp-cli draft create` 用封面 media_id + 正文 HTML 建草稿；
  5. 把返回的草稿 media_id 反馈给用户。

### 10.2 `INSTALL.md`（面向 agent，机器可执行）

agent 无关，不硬编码特定 agent 的 skills 路径：

1. **前置**：Go 1.22+（编译 CLI）；能访问 Web 服务；已从运维处拿到 `MP_HELPER_API_URL` 与 `MP_HELPER_API_KEY`。
2. **编译 CLI**：`make cli` 或 `go build -o <bindir>/mp-cli ./cmd/mp-cli`；把 `mp-cli` 放到 PATH。
3. **配置环境变量**：导出 `MP_HELPER_API_URL`、`MP_HELPER_API_KEY`。
4. **安装 skill**：把 `skill/mp-helper/`（含 `SKILL.md`）复制到「你所用 agent 的 skills 目录」。给出常见示例而非唯一路径：
   - Claude Code：`~/.claude/skills/mp-helper/` 或项目 `.claude/skills/mp-helper/`；
   - 其它 agent：放到其等价的 skills 目录。
5. **验证**：`mp-cli health` 返回 ok；可选跑一次 `material upload` 与 `draft create` 跑通端到端。

### 10.3 `README.md`（面向人类，部署 Web 服务）

构建 `mp-server`、写 `config.yaml`、初始化 DB、用 `/admin` 接口建账号与 key（含 curl 示例）、systemd/Docker 部署建议、安全注意事项（admin token、文件权限、HTTPS）。

## 11. 测试策略（诚实说明）

- **store**：用临时 SQLite 文件做真实读写测试（账号/key CRUD、按哈希解析、级联删除、唯一约束）。
- **api（HTTP 层）**：用 `httptest` 起 Gin，mock `wechat.manager` 接口，覆盖鉴权（数据面 key / 管理面 token）、参数校验、错误响应、admin CRUD。
- **CLI**：用 `httptest` 起 mock 服务，验证命令 → HTTP 请求映射、`--cover` 文件自动上传、输出格式与退出码。
- **wechat 封装层**：silenceper 直连 `api.weixin.qq.com`，无法纯单测。提供**需真实凭证、CI 默认跳过**（`-tags integration` 或 env gate）的集成测试覆盖 `AddMaterial`/`AddDraft`。此局限在文档与 spec 中明确写出，不假装覆盖。

## 12. 构建与部署

- `Makefile`：`make build`（双二进制）、`make server`、`make cli`、`make test`、`make lint`。
- 纯 Go 依赖（含 sqlite 驱动），`CGO_ENABLED=0` 可静态交叉编译。
- 部署：`mp-server --config config.yaml`；建议置于 HTTPS 反代之后；SQLite 文件与 config 文件权限收紧（600）。

## 13. 已解决的关键决策记录

1. 语言/SDK：Go + silenceper/wechat v2（替代最初的 python-weixin 方案）。
2. Web 框架 Gin；仓库单 module + `cmd/` 双二进制。
3. 多租户：1 个 API Key 绑 1 个公众号，key 即鉴权即选账号。
4. 数据存储 SQLite（`modernc.org/sqlite` 纯 Go），账号与 key 入库；运行配置仍用 YAML（端口等）。
5. 管理方式：带 admin token 的 `/admin` HTTP 接口（不在 CLI 内）。
6. Skill 为通用 agent skill；二进制由源码 `go build`；提供面向 agent 的 `INSTALL.md`。

## 14. 后续可选增强（非本次范围）

- app_secret 主密钥加密存储。
- 多 worker 共享 token（Redis，已预留配置）。
- 语音/视频素材、`freepublish` 群发。
- 预编译 release 二进制分发。
- CLI 内置 admin 子命令（包装 `/admin` 接口）。

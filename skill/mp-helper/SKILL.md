---
name: mp-helper
description: >-
  微信公众号（WeChat Official Account / 公众号）图文草稿与图片素材助手——把文章发布到
  公众号草稿箱、上传公众号图片/封面素材。只要用户想「发/推送/同步文章到微信公众号」
  「放进或生成公众号草稿（草稿箱）」「把（AI 生成或已有的）文章排版成公众号图文待发」
  「上传公众号封面或正文配图、要 media_id 或可在正文引用的图片 URL」，就用本 skill——
  哪怕没明说「草稿」「素材」或工具名。能力：① 上传永久图片素材（得 media_id 作封面 +
  url 供正文 <img> 引用）；② 用 标题+正文HTML+封面 创建图文草稿（得草稿 media_id，进
  草稿箱）；③ 回读草稿箱、查询已发布图文与数据统计（只读）；④ 公众号数据分析：单篇文章表现（阅读、来源、分享、收藏、7 天数据、推送打开率、送达人数）、每日涨粉掉粉与发文节奏、推送时段对比；⑤ 健康检查。一把 API Key 对应一个公众号。不要用于：直接群发/正式发布给粉丝
  （本 skill 只到草稿箱）、视频/语音素材、公众号菜单/自动回复/客服/模板消息/IP白名单等
  配置、微信个人号或好友群发、其它平台（微博/小红书）、微信小程序/企业微信/微信支付。
version: 1.4.0
required_environment_variables:
  - name: MP_HELPER_API_URL
    prompt: mp-helper 服务地址（mp-server 的 https 地址）
    help: 例如 https://mp-helper.aworld.ltd
    required_for: 全部功能
  - name: MP_HELPER_API_KEY
    prompt: mp-helper 业务 API Key（一把 key 对应一个公众号）
    help: 由运维用 admin 接口签发；服务部署在 https://mp-helper.aworld.ltd
    required_for: 全部功能
---

# mp-helper —— 微信公众号素材与草稿助手

通过命令行工具 `mp-cli` 把图片上传为公众号永久素材，并把图文文章创建到公众号**草稿箱**。
`mp-cli` 调用部署在服务器上的 `mp-server`（封装微信公众平台 API，服务端管理公众号
appid/secret 与 access_token）。

## 能做什么（功能）

| 功能 | 命令 | 输入 | 输出 |
|---|---|---|---|
| 上传图片素材 | `mp-cli material upload <file>` | 本地图片 | `media_id`（可作封面）+ `url`（可嵌正文 `<img>`） |
| 创建图文草稿 | `mp-cli draft create ...` | 标题 + 正文 HTML + 封面 | 草稿 `media_id`（进入草稿箱） |
| 上传正文图片（原字节） | `mp-cli body-image upload <file>` | jpg/png，小于 1 MB | 正文图片 `url`（只能嵌正文 `<img>`，不能作封面；不占永久素材库） |
| 更新已有草稿 | `mp-cli draft update --media-id <id> --index 0 --article-file a.json` | 草稿 ID + 篇序 + 完整文章 JSON | 同一草稿 `media_id`（原地覆盖该篇，不新建） |
| 列出 / 读取草稿 | `mp-cli draft list`、`mp-cli draft get <media_id>` | 无 / 草稿 `media_id` | 草稿箱原始返回：保存后回读核对标题、正文、封面是否与提交一致 |
| 查询已发布图文 | `mp-cli published list`、`mp-cli published get <article_id>` | 无 / `article_id` | 已发布列表与单篇（含正文）；「群发」的图文不在列表里 |
| 数据统计 | `mp-cli stats <api> --begin --end` | 接口名 + 日期 | 微信原始统计：单篇逐日明细（含群发）、单篇每日阅读 / 分享、每日阅读汇总、用户增减、累计用户 |
| 文章表现分析 | `mp-cli analytics articles --begin --end [--format csv]` | 发表日期区间 | 每篇一行：当天 / 7 天 / 最新的阅读、来源、分享、收藏，推送打开率、推算送达人数、推送时刻 |
| 涨粉掉粉分析 | `mp-cli analytics daily --begin --end [--format csv]` | 日期区间 | 每天一行：新增、取关、累计、取关率、阅读与来源、推送篇数、日类型；附推送日 / 次日 / 不发日平均 |
| 补录推送时刻 / 分类 | `mp-cli article meta set <msgid> --published-at "YYYY-MM-DD HH:MM"` | msgid + 时刻 / 分类 / 备注 | 补录结果（只存本服务，不改公众号） |
| 健康检查 | `mp-cli health` | 无 | `{"status":"ok"}` |

- **永久图片素材**：上传后既得 `media_id`（用作草稿封面 `thumb_media_id`），又得 `url`（用于正文里引用图片）。
- **多公众号**：一把 `MP_HELPER_API_KEY` 唯一对应一个公众号；换公众号就换 key。
- **正文图片 vs 永久素材**：`body-image upload` 走微信「上传图文消息内的图片」接口，服务端原样转发文件字节，不缩放不重编码；只接受 jpg/png 且小于 1 MB，超了直接拒绝，不会替你压缩。封面仍用 `material upload` 得到的 `media_id`。
- **更新草稿**：`draft update` 只覆盖指定草稿 ID 的指定篇（`--index` 必须显式给，0 为首篇），文章 JSON 必须写全 `title author digest content content_source_url thumb_media_id show_cover_pic need_open_comment only_fans_can_comment` 九个字段（空值也要写出来），失败不会改为新建、也不自动重试；超时或 5xx 表示结果不确定，先 `draft get` 回读再决定。

## 能帮用户实现什么

- 把「AI 生成或用户提供的文章」排版成 HTML，连同封面图，一键放进公众号**草稿箱**，作者再到公众号后台审阅、手动群发。
- 批量上传公众号配图，拿到可在正文中引用的图片 URL。
- 作为公众号发文流水线的「最后一公里」：内容生成 → 上传配图/封面 → 建草稿。

## 何时使用本 skill

用户出现下列意图之一时应使用：

- “发到 / 推送到微信公众号”“放进公众号草稿箱”“公众号发文 / 排版图文”
- “上传公众号图片 / 封面素材”“要一个能在公众号正文里用的图片 URL”
- 把一篇（已有或刚生成的）文章准备成公众号图文、生成草稿待发

- 「哪篇 / 哪类文章更受欢迎」「几点发更好」「涨粉掉粉跟发文有没有关系」「隔天发和每天发哪个掉粉少」这类公众号数据分析

## 何时不要用（超出范围）

- **直接群发 / 正式发布**给粉丝：本 skill 只创建到草稿箱为止；群发需作者在公众号后台手动确认（合规与安全考虑）。
- 视频 / 语音素材：当前仅支持**图片**。
- 公众号自定义菜单、自动回复、客服消息、模板/订阅消息、用户/粉丝管理：均不在范围。
- 微信小程序、企业微信、微信支付：不支持。

## 前置条件

- `mp-cli` 已在 PATH。若没有，下载对应平台预编译二进制（无需 Go；详见 `INSTALL.md`）：
  - Linux x86_64：`https://mp-helper.aworld.ltd/download/mp-cli-linux-amd64`
  - macOS Apple Silicon：`https://mp-helper.aworld.ltd/download/mp-cli-darwin-arm64`
  - Windows x86_64：`https://mp-helper.aworld.ltd/download/mp-cli-windows-amd64.exe`

  ```bash
  # Linux/macOS 自动识别并安装（免 sudo，适配沙箱）
  base=https://mp-helper.aworld.ltd/download
  case "$(uname -s)-$(uname -m)" in
    Linux-x86_64) f=mp-cli-linux-amd64;; Darwin-arm64) f=mp-cli-darwin-arm64;;
  esac
  mkdir -p "$HOME/.local/bin"
  curl -fsSL -o "$HOME/.local/bin/mp-cli" "$base/$f" && chmod +x "$HOME/.local/bin/mp-cli"
  export PATH="$HOME/.local/bin:$PATH"   # 确保在 PATH（写进 profile 持久化）
  ```
- 环境变量已配置：
  - `MP_HELPER_API_URL`：mp-server 地址，如 `https://mp-helper.aworld.ltd`
  - `MP_HELPER_API_KEY`：业务 API Key（一把 key 对应一个公众号）

## 标准工作流

1. 自检服务可用：
   ```bash
   mp-cli health
   ```
2. 上传封面图，拿到 `media_id`（作草稿封面）：
   ```bash
   mp-cli material upload ./cover.png
   # => {"media_id":"...","url":"..."}
   ```
3. 若正文需要配图，同样上传，拿到 `url`，把 `url` 写进正文 HTML 的 `<img src="...">`。
4. 把正文写入一个 HTML 文件（如 `article.html`），创建草稿：
   ```bash
   mp-cli draft create \
     --title "标题" \
     --content-file ./article.html \
     --cover <封面 media_id 或 ./cover.png> \
     --author "作者" --digest "摘要"
   # => {"media_id":"草稿 media_id"}
   ```
   `--cover` 传本地图片路径时会自动先上传再用其 media_id。
5. 回读核对：
   ```bash
   mp-cli draft get <草稿 media_id>
   ```
   对照提交的标题、正文、封面，确认没有丢图、错序或样式变化。
6. 把返回的草稿 `media_id` 与回读结果反馈给用户（可在公众号后台「草稿箱」看到，作者审阅后手动群发）。

## 数据分析工作流

用于回答「哪篇文章表现好」「几点推送打开率高」「发文节奏和掉粉的关系」。数据由服务端整理好，直接用，不要自己去拼 `stats` 原始返回。

1. 拉单篇表现（按**发表日期**选区间，每次最多 31 天，最晚到昨天；更长的时间段分几次拉）：
   ```bash
   mp-cli analytics articles --begin 2026-09-11 --end 2026-10-11
   ```
2. 补推送时刻（做「几点发好」时需要）：`notified` 为 true 但 `publish_time` 为 null 的文章（个数见 `missing_publish_time`），把标题和发表日期列给用户，问当时几点推送的，再逐篇补录：
   ```bash
   mp-cli article meta set 2247549594_1 --published-at "2026-10-05 21:00"
   ```
   用户记不清就跳过，分析时说明这些文章没有计入时段对比。
3. 补分类（做「哪类内容受欢迎」时需要）：先问用户用哪几类（如攻略 / 快讯 / 专题）；按标题给每篇拟一个分类，**列表给用户确认或修改后**再逐篇写入，不要未经确认直接写：
   ```bash
   mp-cli article meta set 2247549594_1 --category 攻略
   ```
   补录完重新执行第 1 步，`by_category` 就是按分类的对比。已补录的可用 `mp-cli article meta list` 查看；录错了重新 `set` 覆盖。
4. 拉每日数据（同样最多 31 天一次；更早的 6 天会自动多取，用来判定节奏）：
   ```bash
   mp-cli analytics daily --begin 2026-09-11 --end 2026-10-11
   ```
5. 分析：
   - 哪类内容受欢迎 → `by_category`，以 `median_day7_read` 为主（平均值容易被单篇爆款拉高），再看 `avg_push_open_rate`、分享、收藏。
   - 几点发好 → `by_publish_hour`，比较各小时的 `avg_push_open_rate` / `median_push_open_rate`，同时报每组篇数。
   - 涨粉掉粉与发文 → `summary`（推送日 / 次日 / 不发日）。
   - 隔天发和每天发 → `by_cadence` 与 `cadence_periods`，比较 `daily` 和 `every_other_day` 的平均取关率。
   - 数据多时用 `--format csv` 导出成表格再算（分组与汇总只在 json 里）。

### 字段口径

| 字段 | 含义 |
|---|---|
| `day0` / `day7` / `latest` | 截至发表当天 / 发表日 + 6 天 / 最近一天的**累计**数据；文章未满 7 天时 `day7` 为 null |
| `push_read_user` | 来源为「公众号消息」（即推送）的阅读人数 |
| `read_by_source` | 阅读来源：公众号消息、聊天会话、朋友圈、公众号主页（≈ 历史消息）、推荐、搜一搜、其他 |
| `push_open_rate` | 推送打开率 = 公众号消息阅读人数 ÷ 送达人数（即微信的 `read_delivery_rate`） |
| `delivered_est` | 送达人数**推算值** = 公众号消息阅读人数 ÷ 推送打开率；微信不直接提供 |
| `notified` | true 为推送给粉丝（已通知）；false 为未开启通知，没有打开率和送达人数，不计入推送对比 |
| `publish_time_source` | `manual` 人工补录；`freepublish` 取自发布记录（只在落在发表当天时才采用）；null 未知 |
| `day_type` | `pushed` 当天有推送；`day_after_push` 推送次日；`quiet` 不发日 |
| `cancel_rate` | 当天取关 ÷ 前一天累计关注 |
| `new_by_source` | 新增关注按微信 `user_source` 编码拆分 |
| `summary` | 按 `day_type` 的平均新增、取关、取关率 |
| `by_category` | 按补录分类分组（未补录的归入「未分类」）：篇数、已满 7 天的篇数、7 天阅读的平均与中位数、7 天推送阅读 / 分享 / 收藏平均、推送打开率的平均与中位数 |
| `by_publish_hour` | 按推送时刻的小时（`"00"`–`"23"`）分组，字段同上；只含已通知且有推送时刻的文章 |
| `missing_publish_time` | 已通知但没有推送时刻、未计入 `by_publish_hour` 的篇数 |
| `push_days_7d` / `cadence` | 截至当天近 7 天（含当天）有推送的天数；节奏 `daily`（≥6 天）、`every_other_day`（3–5 天，即隔天发）、`sparse`（1–2 天）、`none`（0 天）；数据不全时为空 |
| `by_cadence` | 按节奏的平均新增、取关、取关率 |
| `cadence_periods` | 节奏相同的连续日期区间及各区间的平均新增、取关、取关率 |

### 必须告诉用户的限制

- **没有按小时的数据**：微信已下线分时接口，看不了粉丝一天里几点在读。「几点发好」只能拿补录的推送时刻对比各篇的 `push_open_rate`，样本少时结论只能参考。
- 取关无法归到具体某篇文章，`daily` 只能看相关性；隔天发和每天发要各积累几周数据才有比较意义，`by_cadence` 里某类只有几天时要明说样本太少。
- 分组对比要同时报篇数 / 天数；某组只有一两篇时不要下结论。
- 统计数据从 2025-11-01 开始；每篇只统计发表后 30 天；次日 8 点后再查前一天，`is_delay` 为 true 表示数据还没出全。
- 返回里的 `warnings` 是个别日期取数失败或还没出数，其余数据照常可用，要在结论里提到。

## 命令与参数速查

```
mp-cli health
mp-cli material upload <file> [--type image|thumb]
mp-cli draft create
    --title <标题>            (必填)
    --content-file <a.html>   (必填，正文 HTML 文件)
    --cover <media_id|图片路径> (必填，传图片则自动先上传)
    [--author <作者>] [--digest <摘要>] [--source-url <原文链接>]
    [--show-cover-pic] [--need-open-comment] [--only-fans-can-comment]
mp-cli draft list [--offset 0] [--count 20] [--with-content]
mp-cli draft get <media_id>
mp-cli draft update --media-id <草稿 media_id> --index <篇序，0 起> --article-file <article.json>
mp-cli body-image upload <file>   (jpg/png，小于 1 MB，返回正文图片 url)
mp-cli published list [--offset 0] [--count 20] [--with-content]
mp-cli published get <article_id>
mp-cli stats <getarticletotaldetail|getarticleread|getarticleshare|getbizsummary|getusersummary|getusercumulate> --begin YYYY-MM-DD --end YYYY-MM-DD
mp-cli analytics articles --begin YYYY-MM-DD --end YYYY-MM-DD [--series] [--format json|csv]
mp-cli analytics daily    --begin YYYY-MM-DD --end YYYY-MM-DD [--format json|csv]
    (begin ≥ 2025-11-01，end ≤ 昨天，跨度 ≤ 31 天；csv 模式的告警写到 stderr)
mp-cli article meta set <msgid> [--published-at "YYYY-MM-DD HH:MM"] [--category <分类>] [--note <备注>]
mp-cli article meta list
```

只读命令原样输出微信返回，不改字段名；本 skill 仍不做发布、群发、删除。

## 输出与错误

- 所有命令成功时输出 JSON 到 stdout；失败时退出码非 0，错误写 stderr。
- `401 unauthorized`：API Key 未配置或无效；检查 `MP_HELPER_API_KEY`。
- `502 wechat_error`：微信返回的错误，`message` 含微信 errmsg、`wechat_errcode` 为微信错误码。常见：
  - `40164`：服务器公网 IP 不在该公众号「IP 白名单」中 → 让用户到公众号后台「设置与开发 → 基本配置 → IP白名单」添加报错信息里的 IP。
  - `40001/42001`：access_token 失效/过期（一般服务端会自动重取，重试即可）。
  - `48001`：该公众号没有这个接口的权限（如未认证），属平台所限，不要反复重试。
  - `47009`：微信已下线该接口（旧版图文统计、分时数据），改用 `analytics` 或 `stats getarticletotaldetail`。
  - `45009`：当天接口调用次数用完，明天再试；同一区间重复查询会命中服务端缓存，不额外消耗。
- 图片建议为 png/jpg/jpeg，单张 ≤ 10MB。

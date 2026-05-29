---
name: mp-helper
description: >-
  微信公众号（WeChat Official Account / 公众号）图文草稿与图片素材助手——把文章发布到
  公众号草稿箱、上传公众号图片/封面素材。只要用户想「发/推送/同步文章到微信公众号」
  「放进或生成公众号草稿（草稿箱）」「把（AI 生成或已有的）文章排版成公众号图文待发」
  「上传公众号封面或正文配图、要 media_id 或可在正文引用的图片 URL」，就用本 skill——
  哪怕没明说「草稿」「素材」或工具名。能力：① 上传永久图片素材（得 media_id 作封面 +
  url 供正文 <img> 引用）；② 用 标题+正文HTML+封面 创建图文草稿（得草稿 media_id，进
  草稿箱）；③ 健康检查。一把 API Key 对应一个公众号。不要用于：直接群发/正式发布给粉丝
  （本 skill 只到草稿箱）、视频/语音素材、公众号菜单/自动回复/客服/模板消息/IP白名单等
  配置、微信个人号或好友群发、其它平台（微博/小红书）、微信小程序/企业微信/微信支付。
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
| 健康检查 | `mp-cli health` | 无 | `{"status":"ok"}` |

- **永久图片素材**：上传后既得 `media_id`（用作草稿封面 `thumb_media_id`），又得 `url`（用于正文里引用图片）。
- **多公众号**：一把 `MP_HELPER_API_KEY` 唯一对应一个公众号；换公众号就换 key。

## 能帮用户实现什么

- 把「AI 生成或用户提供的文章」排版成 HTML，连同封面图，一键放进公众号**草稿箱**，作者再到公众号后台审阅、手动群发。
- 批量上传公众号配图，拿到可在正文中引用的图片 URL。
- 作为公众号发文流水线的「最后一公里」：内容生成 → 上传配图/封面 → 建草稿。

## 何时使用本 skill

用户出现下列意图之一时应使用：

- “发到 / 推送到微信公众号”“放进公众号草稿箱”“公众号发文 / 排版图文”
- “上传公众号图片 / 封面素材”“要一个能在公众号正文里用的图片 URL”
- 把一篇（已有或刚生成的）文章准备成公众号图文、生成草稿待发

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
  # Linux/macOS 自动识别并安装
  base=https://mp-helper.aworld.ltd/download
  case "$(uname -s)-$(uname -m)" in
    Linux-x86_64) f=mp-cli-linux-amd64;; Darwin-arm64) f=mp-cli-darwin-arm64;;
  esac
  curl -fsSL -o mp-cli "$base/$f" && chmod +x mp-cli && sudo install mp-cli /usr/local/bin/mp-cli
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
5. 把返回的草稿 `media_id` 反馈给用户（可在公众号后台「草稿箱」看到，作者审阅后手动群发）。

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
```

## 输出与错误

- 所有命令成功时输出 JSON 到 stdout；失败时退出码非 0，错误写 stderr。
- `401 unauthorized`：API Key 未配置或无效；检查 `MP_HELPER_API_KEY`。
- `502 wechat_error`：微信返回的错误，`message` 含微信 errmsg、`wechat_errcode` 为微信错误码。常见：
  - `40164`：服务器公网 IP 不在该公众号「IP 白名单」中 → 让用户到公众号后台「设置与开发 → 基本配置 → IP白名单」添加报错信息里的 IP。
  - `40001/42001`：access_token 失效/过期（一般服务端会自动重取，重试即可）。
- 图片建议为 png/jpg/jpeg，单张 ≤ 10MB。

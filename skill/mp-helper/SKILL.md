---
name: mp-helper
description: Use when the user wants to upload images to a WeChat Official Account or create a WeChat draft (草稿箱) article. Wraps the `mp-cli` tool which talks to a self-hosted mp-server.
---

# mp-helper

通过 `mp-cli` 上传微信公众号图片素材并创建图文草稿。`mp-cli` 调用部署在服务器上的 `mp-server`。

## 前置条件

- `mp-cli` 已在 PATH（安装见仓库 `INSTALL.md`）。
- 环境变量已配置：
  - `MP_HELPER_API_URL`：mp-server 地址，如 `https://mp.example.com`
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
5. 把返回的草稿 `media_id` 反馈给用户（可在公众号后台「草稿箱」看到）。

## 输出与错误

- 所有命令输出 JSON。失败时退出码非 0，错误写 stderr。
- 401：API Key 未配置或无效；检查 `MP_HELPER_API_KEY`。
- `wechat_error`：微信返回的错误，message 含微信 errmsg。

# 安装 mp-helper skill（照此配置即可）

本文件指导把 `mp-helper` skill 装进 agent。skill 通用、不绑定特定 agent；下方先给「任何 agent 都一样的三件事」，再给 **Hermes**、**Claude Code / 通用 SKILL.md agent**、**纯命令行** 三种具体装法。

## 这个 skill 是什么 / 何时用

让 agent 通过命令行工具 `mp-cli` 操作**微信公众号**：把图片上传为永久素材、把图文文章创建到**草稿箱**。

- **能做**：① 上传图片素材（得 `media_id` 作封面 + `url` 供正文引用）；② 创建图文草稿到草稿箱（得草稿 `media_id`）；③ 健康检查。一把 API Key 对应一个公众号。
- **何时用**：用户想「发/推送到公众号」「放进公众号草稿箱」「上传公众号图片/封面」「把文章排版成公众号图文」时。
- **不适用**：直接群发（仅到草稿箱）；视频/语音素材；公众号菜单/自动回复/客服/模板消息/IP白名单等配置；微信个人号、其他平台、小程序/企业微信/支付。

## 安装总览（任何 agent 都是这三件事）

1. **让 agent 读到 `SKILL.md`**（用于触发）—— 放进该 agent 的 skills 机制。
2. **装 `mp-cli`**（被 skill 调用的命令行）—— 下载对应平台二进制到 PATH。
3. **配 2 个环境变量** —— `MP_HELPER_API_URL`、`MP_HELPER_API_KEY`（一把 key 对应一个公众号）。

SKILL.md 与各平台二进制都可直接从服务域名获取（无需克隆源码、不依赖 GitHub 可见性）：

```
https://mp-helper.aworld.ltd/download/SKILL.md
https://mp-helper.aworld.ltd/download/mp-cli-linux-amd64
https://mp-helper.aworld.ltd/download/mp-cli-darwin-arm64
https://mp-helper.aworld.ltd/download/mp-cli-windows-amd64.exe
https://mp-helper.aworld.ltd/download/sha256sums.txt
```

---

## A. Hermes agent（nousresearch）

Hermes 的技能放在 `~/.hermes/skills/`，支持从 URL 一键安装，并能按 `SKILL.md` 里声明的
`required_environment_variables` 提示/注入环境变量到 terminal、execute_code 沙箱。

**1. 安装 skill（从 URL，一键）**
```bash
hermes skills install https://mp-helper.aworld.ltd/download/SKILL.md --name mp-helper
```
装好后位于 `~/.hermes/skills/.../mp-helper/SKILL.md`，并作为 `/mp-helper` 出现。

**2. 配置环境变量**（skill 已声明 `MP_HELPER_API_URL` / `MP_HELPER_API_KEY`）
- 本地 CLI：首次加载该 skill 时 Hermes 会安全提示你输入；或运行 `hermes setup`；
- 或直接写入 `~/.hermes/.env`：
  ```bash
  MP_HELPER_API_URL=https://mp-helper.aworld.ltd
  MP_HELPER_API_KEY=mpk_xxx      # 你的业务 key
  ```
  这两个变量会被自动注入 Hermes 的 terminal / execute_code 沙箱。

**3. 确保 `mp-cli` 在沙箱可用**
skill 被调用时会按 `SKILL.md` 的说明自动把 `mp-cli` 下载到 `~/.local/bin`（沙箱需有网络与
`curl`）。若沙箱无网络或希望更快，可预先把对应平台的 `mp-cli` 放进沙箱镜像的 PATH。

**4. 验证**
```bash
hermes chat --toolsets skills -q "把 ./cover.png 上传到公众号，并用它当封面建一篇标题为《测试》的草稿"
# 或先自检：
hermes chat --toolsets skills -q "mp-helper 服务通吗"
```

---

## B. Claude Code / 通用「SKILL.md + 目录」型 agent

**1. 装 `mp-cli`（免 sudo）**
```bash
base=https://mp-helper.aworld.ltd/download
case "$(uname -s)-$(uname -m)" in
  Linux-x86_64) f=mp-cli-linux-amd64;; Darwin-arm64) f=mp-cli-darwin-arm64;;
  *) echo "Windows 用 $base/mp-cli-windows-amd64.exe"; exit 0;;
esac
mkdir -p "$HOME/.local/bin"
curl -fsSL -o "$HOME/.local/bin/mp-cli" "$base/$f" && chmod +x "$HOME/.local/bin/mp-cli"
export PATH="$HOME/.local/bin:$PATH"   # 建议写进 ~/.bashrc / ~/.zshrc 持久化
```

**2. 配环境变量**（写进 shell profile）
```bash
export MP_HELPER_API_URL=https://mp-helper.aworld.ltd
export MP_HELPER_API_KEY=mpk_xxx
```

**3. 放 `SKILL.md` 到该 agent 的 skills 目录**
- Claude Code（个人全局）：`~/.claude/skills/mp-helper/`
- Claude Code（项目内）：`<project>/.claude/skills/mp-helper/`
- 其它 agent：其等价的 skills 目录
```bash
mkdir -p ~/.claude/skills/mp-helper
curl -fsSL -o ~/.claude/skills/mp-helper/SKILL.md https://mp-helper.aworld.ltd/download/SKILL.md
```

---

## C. 纯命令行直接用（不接 agent）

完成上面的「装 mp-cli + 配环境变量」后即可：
```bash
mp-cli health
mp-cli material upload ./cover.png
mp-cli draft create --title "标题" --content-file ./article.html --cover ./cover.png
```

---

## 验证

```bash
mp-cli health      # 期望 {"status":"ok"}
```
端到端：`mp-cli material upload <一张图>` 应返回 `media_id` 与 `url`。

## 常见问题

- `401 unauthorized`：`MP_HELPER_API_KEY` 未配置或无效（被吊销/填错）。
- `502 wechat_error` 且 `wechat_errcode=40164`：服务器公网 IP 不在该公众号「IP 白名单」。
  到公众号后台「设置与开发 → 基本配置 → IP白名单」添加报错信息里的 IP。
- 没有业务 key：由运维用 admin 接口签发（带 `Authorization: Bearer <admin_token>`）：
  `curl -X POST https://mp-helper.aworld.ltd/admin/accounts/<id>/keys -H "$ADMIN" -d '{"label":"agent"}'`。

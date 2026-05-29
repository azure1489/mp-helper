# 安装 mp-helper skill（面向 agent）

本文件指导 agent 在本机安装 `mp-helper` skill。skill 通用、不绑定特定 agent 工具。

## 这个 skill 是什么 / 何时用

`mp-helper` 让 agent 通过 `mp-cli` 操作**微信公众号**：把图片上传为永久素材、把图文文章
创建到**草稿箱**。

- **能做**：① 上传图片素材（得 `media_id` 作封面 + `url` 供正文引用）；② 创建图文草稿
  到草稿箱（得草稿 `media_id`）；③ 健康检查。一把 API Key 对应一个公众号。
- **何时用**：用户想「发/推送到微信公众号」「放进公众号草稿箱」「上传公众号图片/封面」
  「把文章排版成公众号图文」时。
- **不适用**：直接群发（仅到草稿箱为止，群发由作者在后台手动确认）；视频/语音素材；
  公众号菜单/自动回复/客服/模板消息；小程序/企业微信/微信支付。

> 完整功能、工作流、命令与错误码说明见 `skill/mp-helper/SKILL.md`（安装后即 agent 运行时读取的文件）。

## 前置条件

- 能访问 mp-server；已从运维处获得：
  - `MP_HELPER_API_URL`（服务地址，如 `https://mp-helper.aworld.ltd`）
  - `MP_HELPER_API_KEY`（业务 API Key）
- 安装方式优先「下载预编译二进制」（无需 Go）；若需从源码编译则要 Go 1.25+。

## 步骤

### 1. 获取 `mp-cli`（二选一）

**方式 A（推荐）：下载对应平台的预编译二进制**

下载地址（按平台选一）：
- Linux x86_64：`https://mp-helper.aworld.ltd/download/mp-cli-linux-amd64`
- macOS Apple Silicon：`https://mp-helper.aworld.ltd/download/mp-cli-darwin-arm64`
- Windows x86_64：`https://mp-helper.aworld.ltd/download/mp-cli-windows-amd64.exe`
- 校验和：`https://mp-helper.aworld.ltd/download/sha256sums.txt`

Linux / macOS 一键（自动识别平台、装到 PATH）：
```bash
set -e
base="https://mp-helper.aworld.ltd/download"
case "$(uname -s)-$(uname -m)" in
  Linux-x86_64)  f=mp-cli-linux-amd64 ;;
  Darwin-arm64)  f=mp-cli-darwin-arm64 ;;
  *) echo "无预编译包，请用方式 B 从源码编译"; exit 1 ;;
esac
curl -fsSL -o mp-cli "$base/$f"
# 可选：校验 sha256
curl -fsSL "$base/sha256sums.txt" | grep "$f" | awk '{print $1"  mp-cli"}' | shasum -a 256 -c -
chmod +x mp-cli
sudo install mp-cli /usr/local/bin/mp-cli   # 或放到 PATH 中的任意目录
```

Windows（PowerShell）：
```powershell
Invoke-WebRequest "https://mp-helper.aworld.ltd/download/mp-cli-windows-amd64.exe" -OutFile mp-cli.exe
# 将 mp-cli.exe 放到 PATH 中的目录
```

**方式 B（备选）：从源码编译**（需 Go 1.25+；go.mod 声明 `go 1.25`，Go 1.21+ 有网络时会自动下载 toolchain）
```bash
git clone https://github.com/azure1489/mp-helper.git && cd mp-helper
make cli                       # 产物 ./bin/mp-cli
sudo install ./bin/mp-cli /usr/local/bin/mp-cli
```

### 2. 配置环境变量（写进 shell profile）

```bash
export MP_HELPER_API_URL="https://mp-helper.aworld.ltd"
export MP_HELPER_API_KEY="mpk_xxx"
```

### 3. 安装 skill 文件

把 `skill/mp-helper/SKILL.md` 复制到「你所用 agent 的 skills 目录」。不同 agent 位置不同，常见示例：
- Claude Code（个人全局）：`~/.claude/skills/mp-helper/`
- Claude Code（项目内）：`<project>/.claude/skills/mp-helper/`
- 其它 agent：复制到其等价的 skills 目录

`SKILL.md` 可直接从服务域名在线获取（无需克隆源码，也不依赖 GitHub 仓库可见性）：
```bash
mkdir -p ~/.claude/skills/mp-helper
curl -fsSL -o ~/.claude/skills/mp-helper/SKILL.md \
  https://mp-helper.aworld.ltd/download/SKILL.md
```

### 4. 验证

```bash
mp-cli health      # 期望输出 {"status":"ok"}
```
端到端验证：`mp-cli material upload <一张图>` 应返回 `media_id` 与 `url`。

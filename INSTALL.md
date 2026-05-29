# 安装 mp-helper skill（面向 agent）

本文件指导 agent 在本机安装 `mp-helper` skill。skill 通用、不绑定特定 agent 工具。

## 前置条件

- 已安装 Go 1.25+（go.mod 声明 `go 1.25`；若本机是 Go 1.21+，在有网络时 Go 会自动下载所需 toolchain）：`go version`
- 能访问 mp-server；已从运维处获得：
  - `MP_HELPER_API_URL`（服务地址）
  - `MP_HELPER_API_KEY`（业务 API Key）

## 步骤

1. 获取源码（若尚未在本机）：
   ```bash
   git clone https://github.com/azure1489/mp-helper.git
   cd mp-helper
   ```

2. 编译 CLI：
   ```bash
   make cli          # 产物在 ./bin/mp-cli
   # 或：go build -o ./bin/mp-cli ./cmd/mp-cli
   ```

3. 把 `mp-cli` 放进 PATH（任选其一）：
   ```bash
   sudo install ./bin/mp-cli /usr/local/bin/mp-cli
   # 或把 ./bin 加入 PATH
   ```

4. 配置环境变量（写进你的 shell profile）：
   ```bash
   export MP_HELPER_API_URL="https://mp.example.com"
   export MP_HELPER_API_KEY="mpk_xxx"
   ```

5. 安装 skill：把 `skill/mp-helper/`（含 `SKILL.md`）复制到「你所用 agent 的 skills 目录」。不同 agent 位置不同，常见示例：
   - Claude Code（个人全局）：`~/.claude/skills/mp-helper/`
   - Claude Code（项目内）：`<project>/.claude/skills/mp-helper/`
   - 其它 agent：复制到其等价的 skills 目录
   ```bash
   mkdir -p ~/.claude/skills/mp-helper
   cp skill/mp-helper/SKILL.md ~/.claude/skills/mp-helper/
   ```

6. 验证：
   ```bash
   mp-cli health      # 期望输出 {"status":"ok"}
   ```
   如需端到端验证：`mp-cli material upload <一张图>` 应返回 `media_id`。

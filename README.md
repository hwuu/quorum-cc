# quorum-cc

Quorum for Claude Code — 多模型交叉评审插件。通过 MCP 协议将 [OpenCode](https://opencode.ai) 后端接入 [Claude Code](https://docs.anthropic.com/en/docs/claude-code)，实现多模型独立评审。

## 功能

- MCP Server（stdio 模式），Claude Code 自动调用
- 支持多后端并行评审（默认 GLM-5 + MiniMax-M2.5）
- 部分后端失败时返回成功结果 + 错误信息，不阻塞整体
- 可自定义评审 Prompt 模板
- 一键 init：检测环境、生成配置、注册 MCP Server

## 架构

```
Claude Code (Opus 4.6)
    │
    │ MCP (stdio)
    ▼
quorum-cc serve
    │          │
    ▼          ▼
OpenCode     OpenCode
(GLM-5)      (MiniMax-M2.5)
    │          │
    ▼          ▼
结构化评审反馈返回 Claude Code
```

## 安装

```bash
curl -fsSL https://github.com/hwuu/quorum-cc/releases/latest/download/install.sh | bash
```

安装脚本会自动配置 bash/zsh 命令补全。

或从 [Releases](https://github.com/hwuu/quorum-cc/releases) 下载对应平台二进制。

## 前置条件

- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) 已安装
- [OpenCode](https://opencode.ai) 已安装，且已配置好模型 API key（`opencode auth`）

## 使用

### 初始化

```bash
quorum-cc init
```

检测环境（opencode、claude）、生成配置文件、注册 MCP Server 到 `~/.claude.json`。

### 在 Claude Code 中使用

重启 Claude Code 后，直接用自然语言触发评审：

```
> 用 quorum 评审一下当前文件

> 用 glm-5 评审一下这个函数的安全性

> 这是支付模块的核心函数，用 quorum 评审一下，重点关注并发安全
```

### 查看状态

```bash
quorum-cc status
```

### 测试连通性

```bash
quorum-cc test
```

逐个调用已配置的后端，验证端到端连通性。

### 运维命令

```bash
quorum-cc version    # 显示版本信息
quorum-cc status     # 检查 opencode、配置、MCP 注册状态
quorum-cc test       # 逐个测试后端连通性
quorum-cc serve      # 启动 MCP Server（由 Claude Code 自动调用，通常无需手动执行）
```

## 配置

配置文件：`~/.config/quorum-cc/config.yaml`

```yaml
version: "1"
backends:
  GLM-5:
    model: "siliconflow-cn/Pro/zai-org/GLM-5"
    timeout: 300
  MiniMax-M2.5:
    model: "siliconflow-cn/Pro/MiniMaxAI/MiniMax-M2.5"
    timeout: 300
defaults:
  backend: "all"
```

### 新增后端

编辑 `config.yaml` 添加即可，无需重新 init：

```yaml
backends:
  DeepSeek-V3:
    model: "siliconflow-cn/Pro/deepseek-ai/DeepSeek-V3"
    timeout: 300
```

### 自定义评审 Prompt

```yaml
prompt_template: |
  你是安全评审专家。请重点关注 OWASP Top 10 漏洞。
  {{.ContextSection}}
  待评审内容：
  ```
  {{.Content}}
  ```
  输出格式：严重程度 + 问题描述 + 修复建议
```

模板变量：`{{.Content}}`（待评审内容）、`{{.Context}}`（业务上下文）、`{{.ContextSection}}`（格式化的上下文段落）、`{{.FilePath}}`（文件路径）。

## 开发

### 本地构建

```bash
# 构建（版本 tag 自动取当前分支名）
make build

# 指定版本构建
make VERSION=0.1.0 build
```

### 测试

```bash
# 单元测试
make test

# 或直接
go test ./... -count=1
```

## License

MIT

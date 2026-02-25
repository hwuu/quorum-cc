# quorum-cc — Quorum for Claude Code 设计文档

## 目录

- [1. 背景与目标](#1-背景与目标)
  - [1.1 问题](#11-问题)
  - [1.2 启发](#12-启发)
  - [1.3 目标](#13-目标)
  - [1.4 非目标](#14-非目标)
- [2. 总体设计](#2-总体设计)
- [3. 设计决策](#3-设计决策)
  - [3.1 集成方式：为什么选 MCP Server 而非 Shell 脚本](#31-集成方式为什么选-mcp-server-而非-shell-脚本)
  - [3.2 调用方式：为什么选 opencode run 而非 HTTP API](#32-调用方式为什么选-opencode-run-而非-http-api)
  - [3.3 实现语言：为什么选 Go](#33-实现语言为什么选-go)
  - [3.4 MVP 范围：为什么先做单工具而非完整工作流](#34-mvp-范围为什么先做单工具而非完整工作流)
- [4. 架构设计](#4-架构设计)
  - [4.1 核心分层](#41-核心分层)
  - [4.2 数据流](#42-数据流)
- [5. 组件设计](#5-组件设计)
  - [5.1 MCP Server](#51-mcp-server)
  - [5.2 Backend Adapter](#52-backend-adapter)
  - [5.3 CLI 工具](#53-cli-工具)
- [6. 用户体验流程](#6-用户体验流程)
  - [6.1 安装](#61-安装)
  - [6.2 配置](#62-配置)
  - [6.3 日常使用](#63-日常使用)
- [7. 实现规划](#7-实现规划)
  - [7.1 目录结构](#71-目录结构)
  - [7.2 实现步骤](#72-实现步骤)
  - [7.3 测试要点](#73-测试要点)
- [8. 升级机制](#8-升级机制)
- [参考文献](#参考文献)

---

## 1. 背景与目标

### 1.1 问题

使用 Claude Code (opus 4.6) 做项目时，存在以下局限：

| 局限 | 说明 |
|------|------|
| **单模型盲区** | 单一模型存在认知偏见，容易对自身输出过度自信 |
| **缺乏交叉验证** | Claude Code 自审自己的代码/设计，难以发现系统性盲点 |
| **评审流程手动** | 需要手动复制代码到其他工具评审，上下文切换成本高 |
| **闲置模型资源** | 本地已安装 OpenCode (glm-5, minimax m2.5)，但无法与 Claude Code 协作 |

### 1.2 启发

- **SWE-Debate**（上海交大+华为, ICSE 2026）：竞争性多智能体辩论比单智能体自反思更能发现问题
- **AI Red Teaming**（Microsoft, OpenAI）：独立评审者（红队）能发现主模型的系统性盲区
- **oh-my-opencode**：多模型协作的工程实践，但以 OpenCode 为中心取代 Claude Code，方向相反

### 1.3 目标

构建一个 Claude Code 插件 **quorum-cc**，通过 MCP 协议将 OpenCode 后端接入 Claude Code：

- **一键配置**：`curl -fsSL .../install.sh | bash && quorum-cc init`，自动配置 Claude Code MCP
- **统一入口**：所有操作在 Claude Code 内完成，无需切换窗口
- **多模型评审**：Claude Code 生成 → OpenCode 后端独立评审 → 结构化反馈
- **后端可插拔**：支持 glm-5、minimax m2.5，未来可扩展其他模型

### 1.4 非目标

- **不做工作流引擎**：不内置 Idea→Design→Code 阶段管理，由用户/CLAUDE.md 控制
- **不做状态持久化**：MVP 不做 Ralph Loop 状态机、ADR 自动生成
- **不做共识引擎**：MVP 不做多模型评分对比和冲突检测，返回原始评审结果即可
- **不取代 Claude Code**：增强而非替代，Claude Code 始终是主控

---

## 2. 总体设计

核心思路：**MCP Server + OpenCode CLI 后端**。Claude Code 通过 MCP 协议调用本地 MCP Server，Server 将请求转发给 OpenCode CLI (`opencode run`)，返回结构化评审结果。

```
+------------------------------------------------------------------+
|                        Claude Code (opus 4.6)                     |
|                           Main Controller                         |
|                                                                   |
|  User: "review this code with opencode"                          |
|       |                                                           |
|       v                                                           |
|  MCP Client ----(stdio)----> quorum-cc MCP Server                |
|                              |                                    |
+------------------------------+------------------------------------+
                               |
                               v
                  +---------------------------+
                  |    quorum-cc MCP Server    |
                  |    (localhost, stdio)      |
                  |                            |
                  |  Tools:                    |
                  |  - quorum_review           |
                  +---------------------------+
                       |              |
                       v              v
              +--------------+  +--------------+
              | opencode run |  | opencode run |
              | -m glm-5     |  | -m minimax   |
              +--------------+  +--------------+
```

关键设计决策：

| 决策 | 选择 | 核心理由 | 详见 |
|------|------|----------|------|
| 集成方式 | MCP Server | Claude Code 原生支持，无需 hack | [3.1](#31-集成方式为什么选-mcp-server-而非-shell-脚本) |
| 调用方式 | opencode run | 非交互式，直接可用 | [3.2](#32-调用方式为什么选-opencode-run-而非-http-api) |
| 实现语言 | Go | 单二进制分发，零依赖 | [3.3](#33-实现语言为什么选-go) |
| MVP 范围 | 单工具 review | 先验证核心价值再扩展 | [3.4](#34-mvp-范围为什么先做单工具而非完整工作流) |

---

## 3. 设计决策

### 3.1 集成方式：为什么选 MCP Server 而非 Shell 脚本

| 维度 | MCP Server | Shell 脚本 (Bash Hook) |
|------|-----------|----------------------|
| Claude Code 集成 | 原生支持，工具调用 | 需要 hook 或手动触发 |
| 参数传递 | 结构化 JSON | 环境变量/临时文件 |
| 返回值 | 结构化文本，Claude 可直接理解 | stdout 文本，需要约定格式 |
| 错误处理 | MCP 协议内置 | exit code + stderr |
| 用户体验 | Claude 自动调用，无感知 | 需要用户手动触发或配置 hook |

**决策**：选择 MCP Server。

**理由**：
1. Claude Code 原生支持 MCP，工具调用是一等公民
2. 结构化输入输出，Claude 能直接理解评审结果并据此行动
3. 用户在 Claude Code 内说"帮我评审这段代码"即可触发，无需切换窗口

**代价**：
1. 需要实现 MCP 协议（但 mcp-go SDK 已封装好）
2. 需要常驻进程（但 stdio 模式由 Claude Code 按需启动）

### 3.2 调用方式：为什么选 opencode run 而非 HTTP API

| 维度 | opencode run (CLI) | HTTP API (opencode serve) |
|------|-------------------|--------------------------|
| 依赖 | 仅需 opencode 已安装 | 需要启动 server 进程 |
| 配置 | 零配置，直接调用 | 需要管理端口、认证 |
| 模型切换 | `-m provider/model` 参数 | 需要 API 参数或多实例 |
| 并发 | 多进程并行 | 单 server 串行或需队列 |
| 超时控制 | subprocess timeout | HTTP timeout |

**决策**：选择 `opencode run`。

**理由**：
1. 零额外配置，用户装好 opencode 就能用
2. `-m` 参数天然支持多模型切换
3. 并行评审只需同时启动多个子进程

**代价**：
1. 每次调用有进程启动开销（约 1-2 秒）
2. 无法复用会话上下文（每次 run 是独立的）

### 3.3 实现语言：为什么选 Go

| 维度 | Go | Python | TypeScript |
|------|-----|--------|-----------|
| 分发方式 | 单二进制，下载即用 | `pip install`，需 Python 3.10+ | `npm install`，需 Node.js |
| MCP SDK | `mcp-go`（社区，star 5k+，广泛使用） | `mcp` 官方 SDK，成熟 | `@modelcontextprotocol/sdk`，成熟 |
| 无 sudo 环境 | 放 `~/bin` 即可 | `pip install --user` 可能有 PATH 问题 | 全局安装需权限 |
| subprocess 调用 | `os/exec`，原生 | `asyncio.subprocess`，原生 | `child_process`，原生 |
| 并发 | goroutine + errgroup | asyncio.gather | Promise.all |
| 跨平台构建 | goreleaser 一次出四个平台 | 依赖用户环境 | 需要 pkg 或类似工具 |
| 项目一致性 | cloudcode 项目已用 Go | — | — |

**决策**：选择 Go。

**理由**：
1. 单二进制分发，用户 `curl` 下载即可使用，零运行时依赖
2. `mcp-go` 已足够成熟，本项目 MCP 交互简单（一个 tool，stdio 通信）
3. goroutine 天然支持并行调用多个 opencode 后端
4. 与 cloudcode 项目技术栈一致，可复用经验

**代价**：
1. MCP Go SDK 非官方维护，需关注兼容性
2. 开发速度略慢于 Python

### 3.4 MVP 范围：为什么先做单工具而非完整工作流

| 维度 | MVP（单工具 review） | 完整工作流 |
|------|---------------------|-----------|
| 开发量 | 1 个 MCP tool | 5+ tools + 状态机 + 持久化 |
| 验证周期 | 可立即验证核心价值 | 需要完整项目才能验证 |
| 用户学习成本 | 几乎为零 | 需要理解阶段、ADR、Ralph Loop |
| 风险 | 低（不好用就删掉） | 高（大量代码可能浪费） |

**决策**：MVP 只做一个 `quorum_review` 工具。

**理由**：
1. 核心假设是"多模型评审有价值"，一个 review 工具就能验证
2. 如果评审反馈质量不够（弱模型噪音太多），整个项目方向需要调整
3. 工作流、状态机等可以在核心价值验证后逐步叠加

**代价**：
1. 用户需要自己管理评审流程（何时触发、如何处理反馈）
2. 无自动化的 Ralph Loop 收敛检测

---

## 4. 架构设计

### 4.1 核心分层

```
+------------------------------------------------------------------+
|                       quorum-cc Architecture                      |
|                                                                   |
|  +------------------------------------------------------------+  |
|  |  Interface Layer                                            |  |
|  |  +------------------------------------------------------+  |  |
|  |  |  MCP Server (stdio)                                  |  |  |
|  |  |  - Receive tool calls from Claude Code               |  |  |
|  |  |  - Return structured review results                  |  |  |
|  |  |  - Protocol: MCP over stdio                          |  |  |
|  |  +------------------------------------------------------+  |  |
|  +------------------------------------------------------------+  |
|                                                                   |
|  +------------------------------------------------------------+  |
|  |  Core Layer                                                 |  |
|  |  +------------------------------------------------------+  |  |
|  |  |  Review Dispatcher                                   |  |  |
|  |  |  - Parse review request                              |  |  |
|  |  |  - Route to backend(s)                               |  |  |
|  |  |  - Aggregate results                                 |  |  |
|  |  +------------------------------------------------------+  |  |
|  +------------------------------------------------------------+  |
|                                                                   |
|  +------------------------------------------------------------+  |
|  |  Backend Layer                                              |  |
|  |  +------------------------------------------------------+  |  |
|  |  |  OpenCode Adapter                                    |  |  |
|  |  |  - Build review prompt                               |  |  |
|  |  |  - Call `opencode run -m <model>` subprocess         |  |  |
|  |  |  - Parse output                                      |  |  |
|  |  +------------------------------------------------------+  |  |
|  +------------------------------------------------------------+  |
|                                                                   |
|  +------------------------------------------------------------+  |
|  |  Config Layer                                               |  |
|  |  +------------------------------------------------------+  |  |
|  |  |  ~/.config/quorum-cc/config.yaml                     |  |  |
|  |  |  - Backend definitions (model, timeout)              |  |  |
|  |  |  - Default review parameters                         |  |  |
|  |  +------------------------------------------------------+  |  |
|  +------------------------------------------------------------+  |
+------------------------------------------------------------------+
```

### 4.2 数据流

```
Claude Code                quorum-cc MCP Server              OpenCode CLI
    |                              |                              |
    |  MCP tool_call:              |                              |
    |  quorum_review({             |                              |
    |    content: "def foo()...",  |                              |
    |    context: "payment svc",   |                              |
    |    backend: "glm-5"          |                              |
    |  })                          |                              |
    |----------------------------->|                              |
    |                              |  Build review prompt         |
    |                              |  from template               |
    |                              |                              |
    |                              |  subprocess:                 |
    |                              |  opencode run -m glm-5       |
    |                              |    --prompt "<review_prompt>" |
    |                              |----------------------------->|
    |                              |                              |
    |                              |          (model thinking)    |
    |                              |                              |
    |                              |  stdout: review result       |
    |                              |<-----------------------------|
    |                              |                              |
    |  MCP tool_result:            |                              |
    |  { text: "## Review...\n    |                              |
    |    Score: 7/10\n..." }       |                              |
    |<-----------------------------|                              |
    |                              |                              |
    |  Claude reads review,        |                              |
    |  decides next action         |                              |
    |                              |                              |
```

并行评审时，Dispatcher 同时启动多个 subprocess，`errgroup.Group` 并发等待全部完成后合并返回。

**部分失败降级策略：**

并行调用多个后端时，部分后端可能超时或报错。采用"尽力返回"策略：

| 场景 | 行为 |
|------|------|
| 所有后端成功 | 正常返回所有评审结果 |
| 部分后端失败 | 返回成功的结果 + 失败后端的错误信息 |
| 所有后端失败 | 返回错误信息，提示用户检查 opencode 配置 |

部分失败时的返回格式：

```
## GLM-5 Review

(glm-5 评审内容)

---

## MiniMax Review

[ERROR] MiniMax backend timed out after 300s
```

---

## 5. 组件设计

### 5.1 MCP Server

MCP Server 是 quorum-cc 的核心入口，通过 stdio 与 Claude Code 通信。

**暴露的 MCP Tools：**

| Tool 名称 | 描述 | 参数 |
|-----------|------|------|
| `quorum_review` | 将内容发送给 OpenCode 后端进行独立评审 | `content`, `context`, `backend`, `file_path` |

**`quorum_review` 参数定义：**

```json
{
  "type": "object",
  "properties": {
    "content": {
      "type": "string",
      "description": "待评审内容（代码、设计文档等）"
    },
    "context": {
      "type": "string",
      "description": "业务上下文，帮助评审员理解背景（可选）"
    },
    "backend": {
      "type": "string",
      "default": "all",
      "description": "评审后端：配置文件中的后端名称（如 glm-5、minimax），或 all 并行调用所有后端。可用值从 ~/.config/quorum-cc/config.yaml 的 backends 中读取"
    },
    "file_path": {
      "type": "string",
      "description": "文件路径，用于评审报告定位（可选）"
    }
  },
  "required": ["content"]
}
```

**返回格式：**

单后端返回评审文本；`backend: "all"` 时返回多个后端结果的拼接，用分隔线区分：

```
## GLM-5 Review

(glm-5 评审内容)

---

## MiniMax Review

(minimax 评审内容)
```

### 5.2 Backend Adapter

Backend Adapter 负责调用 OpenCode CLI 并返回结果。

**核心逻辑：**

```go
func callOpenCode(ctx context.Context, prompt, model string, timeout time.Duration) (string, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, "opencode", "run", "-m", model, prompt)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("opencode run failed: %s", stderr.String())
    }
    return stdout.String(), nil
}
```

注意：prompt 作为 positional message 传给 `opencode run`，不是通过 `--prompt` 全局选项。使用默认 text 输出格式，直接返回给 Claude Code。

**评审 Prompt 模板：**

```
你是一位独立的代码评审员。请严格评审以下内容，不要客气。

{{.ContextSection}}

待评审内容：
```
{{.Content}}
```

请按以下结构输出：
1. 总体评分 (1-10)
2. 关键发现（按严重程度：Critical / Warning / Info）
3. 改进建议（具体可执行）
```

`{{.ContextSection}}` 在 context 参数非空时渲染为 `业务上下文：<context>`，否则为空。使用 Go 标准库 `text/template` 渲染。

**后端配置：**

```yaml
# ~/.config/quorum-cc/config.yaml
backends:
  glm-5:
    model: "siliconflow-cn/Pro/zai-org/GLM-5"
    timeout: 300
  minimax:
    model: "siliconflow-cn/Pro/MiniMaxAI/MiniMax-M2.5"
    timeout: 300

defaults:
  backend: "all"
```

model 字段的值对应 `opencode run -m` 的参数，格式为 `provider/model`（如 `siliconflow-cn/Pro/zai-org/GLM-5`）。具体值取决于用户的 OpenCode 配置，`quorum-cc init` 时会调用 `opencode models` 检测可用模型并自动填充。

### 5.3 CLI 工具

`quorum-cc` 提供命令行工具用于安装配置，不用于日常评审（日常评审在 Claude Code 内完成）。

**命令设计：**

```
quorum-cc init          # 检测环境 + 配置 Claude Code MCP + 生成 config.yaml
quorum-cc status        # 检查 opencode 可用性、已配置后端、MCP 注册状态
quorum-cc test          # 发送测试评审请求，验证端到端连通性
quorum-cc version       # 显示版本号
```

**`quorum-cc init` 流程：**

```
+----------+     check opencode     +-----------+     detect models     +-----------+
|  Start   | ---------------------> |  Check    | -------------------> |  Generate |
| quorum-cc|                        |  opencode |                      |  config   |
|  init    |                        |  installed|                      |  .yaml    |
+----------+                        +-----------+                      +-----------+
                                                                            |
                                                                            v
+-----------+     output summary     +-----------+     register MCP    +-----------+
|  Done     | <-------------------- |  Verify   | <------------------ |  Update   |
|  Ready!   |                       |  MCP      |                     |  claude   |
|           |                       |  config   |                     |  settings |
+-----------+                       +-----------+                     +-----------+
```

**MCP 注册方式：**

`quorum-cc init` 自动在 `~/.claude.json`（或项目级 `.mcp.json`）中添加：

```json
{
  "mcpServers": {
    "quorum-cc": {
      "command": "quorum-cc",
      "args": ["serve"],
      "description": "Multi-model code review via OpenCode backends"
    }
  }
}
```

`quorum-cc serve` 是 MCP Server 的 stdio 入口，由 Claude Code 按需启动，用户无需手动管理。

---

## 6. 用户体验流程

### 6.1 安装

```bash
# Linux/macOS
curl -fsSL https://github.com/hwuu/quorum-cc/releases/latest/download/install.sh | bash
```

前置条件：
- OpenCode 已安装且可用（`opencode --version` 正常输出）
- OpenCode 已配置好模型 API key（通过 `opencode auth` 配置，`opencode run -m <model> "hello"` 能正常返回）
- Claude Code 已安装

注意：quorum-cc 不管理 API key。它通过子进程调用 `opencode run`，OpenCode 自己负责认证。quorum-cc 的子进程继承当前用户环境，OpenCode 读取自己的凭证配置。

### 6.2 配置

```bash
$ quorum-cc init

quorum-cc — Quorum for Claude Code
===================================

[1/4] Check environment...
  ✓ opencode v0.x.x found at /home/user/.opencode/bin/opencode
  ✓ Claude Code found

[2/4] Detect available models...
  ✓ siliconflow-cn/Pro/zai-org/GLM-5
  ✓ siliconflow-cn/Pro/MiniMaxAI/MiniMax-M2.5

[3/4] Generate config...
  ✓ Created ~/.config/quorum-cc/config.yaml

[4/4] Register MCP server...
  ✓ Updated ~/.claude.json

Done! Restart Claude Code, then try:
  "review this file with quorum"
```

### 6.3 日常使用

所有操作在 Claude Code 内完成，用户通过自然语言触发：

**基础评审（所有后端并行）：**

```
> 用 quorum 评审一下当前文件
```

Claude Code 自动调用 `quorum_review`，读取当前文件内容，发送给所有后端，返回评审结果后自行决定是否修改。

**指定后端：**

```
> 用 glm-5 评审一下这个函数的安全性
```

Claude Code 调用 `quorum_review` 时设置 `backend: "glm-5"`。

**带上下文的评审：**

```
> 这是支付模块的核心函数，用 quorum 评审一下，重点关注并发安全
```

Claude Code 将"支付模块核心函数，重点关注并发安全"作为 `context` 参数传入。

**评审设计文档：**

```
> 用 quorum 评审 docs/design.md
```

Claude Code 读取文件内容，调用 `quorum_review`，返回设计层面的评审意见。

---

## 7. 实现规划

### 7.1 目录结构

```
quorum-cc/
├── cmd/
│   └── quorum-cc/
│       └── main.go              # 入口
├── internal/
│   ├── server/
│   │   └── server.go            # MCP Server (stdio)
│   ├── tools/
│   │   └── review.go            # quorum_review tool 定义与处理
│   ├── dispatcher/
│   │   └── dispatcher.go        # 评审分发：单后端/并行多后端
│   ├── adapter/
│   │   └── opencode.go          # OpenCode CLI 适配器 (os/exec)
│   ├── prompt/
│   │   └── prompt.go            # 评审 Prompt 模板 (text/template)
│   └── config/
│       └── config.go            # 配置文件读写 (YAML)
├── tests/
│   ├── adapter_test.go          # 适配器单元测试（mock subprocess）
│   ├── dispatcher_test.go       # 分发逻辑测试
│   ├── server_test.go           # MCP Server 集成测试
│   └── cli_test.go              # CLI 命令测试
├── install.sh                   # 安装脚本（检测 OS/ARCH，下载二进制）
├── go.mod
├── go.sum
├── Makefile
├── .goreleaser.yml
├── README.md
└── LICENSE
```

### 7.2 实现步骤

| 步骤 | 任务 | 依赖 | 验证方式 |
|------|------|------|----------|
| 1 | Go 项目初始化：go.mod + cobra CLI 框架 | 无 | `go build && quorum-cc --help` |
| 2 | internal/config：配置文件读写 | 步骤 1 | 单元测试通过 |
| 3 | internal/adapter：OpenCode CLI 调用 | 步骤 1 | `opencode run` 能正常调用并返回结果 |
| 4 | internal/prompt：评审 Prompt 模板 | 步骤 1 | 模板渲染输出正确 |
| 5 | internal/dispatcher：单后端/并行分发 | 步骤 3, 4 | 单后端和并行调用均正常 |
| 6 | internal/tools + internal/server：MCP Server | 步骤 5 | `quorum-cc serve` 启动，MCP 协议通信正常 |
| 7 | CLI 命令：init/status/test | 步骤 2, 6 | `quorum-cc init` 端到端配置成功 |
| 8 | goreleaser + install.sh | 步骤 7 | tag 推送后自动发布二进制 |
| 9 | 端到端测试：Claude Code 内调用 | 步骤 8 | 在 Claude Code 中触发评审并收到结果 |

### 7.3 测试要点

| 测试项 | 测试方法 | 验证标准 |
|--------|----------|----------|
| OpenCode 调用 | mock subprocess，验证命令拼接 | 参数正确，timeout 生效 |
| 单后端评审 | 指定 backend=glm-5 | 返回单个评审结果 |
| 并行评审 | backend=all | 两个后端结果均返回，用分隔线区分 |
| 超时处理 | 设置极短 timeout | 超时后返回错误信息而非挂起 |
| OpenCode 不可用 | 卸载 opencode 后调用 | 返回明确错误提示 |
| 空内容 | content 为空字符串 | 返回参数校验错误 |
| 大文件 | 传入超长内容 | 正常处理或截断并提示 |
| init 幂等性 | 重复执行 `quorum-cc init` | 不重复注册 MCP，更新已有配置 |
| MCP 协议 | 用 MCP Inspector 连接 | tool list 和 tool call 均正常 |

---

## 8. 升级机制

### 8.1 quorum-cc 自身升级

```bash
curl -fsSL https://github.com/hwuu/quorum-cc/releases/latest/download/install.sh | bash
```

升级后无需重新 `init`。MCP 注册指向 `quorum-cc serve` 命令，install.sh 会原地替换二进制，Claude Code 下次启动 MCP Server 时自动使用新版本。

### 8.2 配置兼容性

配置文件 `~/.config/quorum-cc/config.yaml` 包含 `version` 字段：

```yaml
version: "1"
backends:
  glm-5:
    model: "siliconflow-cn/Pro/zai-org/GLM-5"
    timeout: 300
```

升级时如果配置格式变更，`quorum-cc serve` 启动时自动迁移（读取旧格式，写入新格式，备份旧文件为 `config.yaml.bak`）。

### 8.3 新增后端

用户手动编辑 `config.yaml` 添加新后端即可，无需重新 init：

```yaml
backends:
  glm-5:
    model: "siliconflow-cn/Pro/zai-org/GLM-5"
    timeout: 300
  minimax:
    model: "siliconflow-cn/Pro/MiniMaxAI/MiniMax-M2.5"
    timeout: 300
  deepseek:                          # 新增
    model: "deepseek/deepseek-chat"
    timeout: 300
```

下次 Claude Code 调用 `quorum_review` 时，`backend: "all"` 会自动包含新后端。

### 8.4 Prompt 模板自定义

用户可在配置中覆盖默认评审 Prompt：

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

不配置则使用内置默认模板。模板语法为 Go `text/template`。

---

## 参考文献

### 核心依赖

- [mcp-go](https://github.com/mark3labs/mcp-go) — MCP Go SDK（社区维护，广泛使用）
- [OpenCode](https://opencode.ai) — AI 编程助手，支持多模型后端
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — Anthropic 官方 CLI
- [Cobra](https://github.com/spf13/cobra) — Go CLI 框架
- [goreleaser](https://goreleaser.com/) — Go 项目发布工具

### 学术参考

- [SWE-Debate](https://arxiv.org/abs/2502.09890) — Competitive Multi-Agent Debate for Software Engineering (ICSE 2026)
- [Multi-Agent Debate](https://arxiv.org/abs/2305.19118) — Encouraging Divergent Thinking in Large Language Models through Multi-Agent Debate

### 相关项目

- [oh-my-opencode](https://github.com/code-yeongyu/oh-my-opencode) — 多模型协作框架（TypeScript + Bun，以 OpenCode 为中心）
- [MCP Servers](https://github.com/modelcontextprotocol/servers) — MCP Server 参考实现集合

---

**文档版本**: 1.4
**更新日期**: 2026-02-24

**修订记录**：
- v1.4: OpenCode 二次 review 修正 — 去掉 `--format json`，使用默认 text 格式避免未解析 JSON 问题；`backend` 参数去掉 enum 硬编码，改为字符串类型支持动态后端扩展
- v1.3: 补充 API key 说明 — 6.1 前置条件明确 OpenCode 需预先配置好模型 API key，quorum-cc 不管理凭证
- v1.2: 实现语言从 Python 改为 Go — 3.3 决策表重写；5.2 代码示例改为 Go；Prompt 模板改为 Go text/template 语法；7.1 目录结构改为 Go 项目布局；7.2 实现步骤增加 goreleaser；安装方式改为 curl + install.sh；参考文献更新为 mcp-go/cobra/goreleaser
- v1.1: OpenCode (GLM-5) review 修正 — `opencode run` 调用语法改为 positional message（非 `--prompt` 全局选项）；模型名称格式修正为实际的 `siliconflow-cn/Pro/...` 格式；补充并行评审部分失败的降级策略；补充 `--format json` 输出格式说明
- v1.0: 初始版本

package tools

import (
	"context"
	"fmt"

	"github.com/hwuu/quorum-cc/internal/config"
	"github.com/hwuu/quorum-cc/internal/dispatcher"
	"github.com/mark3labs/mcp-go/mcp"
)

// NewReviewTool creates the quorum_review MCP tool definition.
func NewReviewTool() mcp.Tool {
	return mcp.NewTool("quorum_review",
		mcp.WithDescription("将内容发送给 OpenCode 后端进行独立评审"),
		mcp.WithString("content",
			mcp.Description("待评审内容（代码、设计文档等），或评审指令（如「评审最近的 git 变更」）。OpenCode 会自行读取文件、执行命令"),
			mcp.Required(),
		),
		mcp.WithString("context",
			mcp.Description("业务上下文，帮助评审员理解背景（可选）"),
		),
		mcp.WithString("backend",
			mcp.Description("评审后端：配置文件中的后端名称（如 glm-5、minimax），或 all 并行调用所有后端"),
			mcp.DefaultString("all"),
		),
		mcp.WithString("file_path",
			mcp.Description("文件路径（可选），作为评审上下文提示"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("超时时间（秒）。覆盖 config 中的默认值。复杂评审建议设置 600-900"),
		),
	)
}

// HandleReview handles the quorum_review tool call.
func HandleReview(cfg *config.Config) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()

		content, _ := args["content"].(string)
		if content == "" {
			return mcp.NewToolResultError("content must not be empty"), nil
		}

		ctxStr, _ := args["context"].(string)
		backend, _ := args["backend"].(string)
		if backend == "" {
			backend = cfg.Defaults.Backend
			if backend == "" {
				backend = "all"
			}
		}
		filePath, _ := args["file_path"].(string)

		var timeoutOverride int
		if t, ok := args["timeout"].(float64); ok && t > 0 {
			timeoutOverride = int(t)
		}

		result, err := dispatcher.Dispatch(ctx, cfg, content, ctxStr, filePath, backend, timeoutOverride)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("review failed: %v", err)), nil
		}

		return mcp.NewToolResultText(result), nil
	}
}

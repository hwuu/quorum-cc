package prompt

import (
	"bytes"
	"strings"
	"text/template"
)

const defaultTemplate = `你是一位独立的代码评审员。请严格评审以下内容，不要客气。
{{.ContextSection}}
{{.Content}}

请按以下结构输出：
1. 总体评分 (1-10)
2. 关键发现（按严重程度：Critical / Warning / Info）
3. 改进建议（具体可执行）`

// Params holds the template parameters.
type Params struct {
	Content        string
	Context        string
	ContextSection string
	FilePath       string
}

// Build renders the review prompt from the given template string and params.
// If tmplStr is empty, the default template is used.
func Build(tmplStr string, content, ctx, filePath string) (string, error) {
	if tmplStr == "" {
		tmplStr = defaultTemplate
	}

	p := Params{
		Content:  content,
		Context:  ctx,
		FilePath: filePath,
	}
	if ctx != "" {
		p.ContextSection = "业务上下文：" + ctx + "\n"
	}
	if filePath != "" {
		p.ContextSection += "文件路径：" + filePath + "\n"
	}

	t, err := template.New("review").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, p); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

package prompt

import (
	"strings"
	"testing"
)

func TestBuildDefault(t *testing.T) {
	result, err := Build("", "func foo() {}", "", "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "func foo() {}") {
		t.Error("result should contain content")
	}
	if !strings.Contains(result, "总体评分") {
		t.Error("result should contain default template structure")
	}
	if strings.Contains(result, "业务上下文") {
		t.Error("result should not contain context section when context is empty")
	}
}

func TestBuildWithContext(t *testing.T) {
	result, err := Build("", "func foo() {}", "支付模块", "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "业务上下文：支付模块") {
		t.Error("result should contain context section")
	}
}

func TestBuildWithFilePath(t *testing.T) {
	result, err := Build("", "func foo() {}", "", "src/auth.go")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "文件路径：src/auth.go") {
		t.Error("result should contain file path")
	}
}

func TestBuildWithContextAndFilePath(t *testing.T) {
	result, err := Build("", "func foo() {}", "支付模块", "src/pay.go")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "业务上下文：支付模块") {
		t.Error("result should contain context")
	}
	if !strings.Contains(result, "文件路径：src/pay.go") {
		t.Error("result should contain file path")
	}
}

func TestBuildCustomTemplate(t *testing.T) {
	tmpl := "Review: {{.Content}} | Context: {{.ContextSection}}"
	result, err := Build(tmpl, "code here", "ctx", "")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "code here") {
		t.Error("result should contain content")
	}
	if !strings.Contains(result, "业务上下文：ctx") {
		t.Error("result should contain context")
	}
}

func TestBuildInvalidTemplate(t *testing.T) {
	_, err := Build("{{.Invalid", "code", "", "")
	if err == nil {
		t.Error("expected error for invalid template")
	}
}

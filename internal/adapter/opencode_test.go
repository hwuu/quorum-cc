package adapter

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestOpenCodeHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
		return
	}
	switch os.Getenv("GO_TEST_HELPER_MODE") {
	case "success":
		os.Stdout.WriteString("## Review\nScore: 8/10\nLooks good.")
		os.Exit(0)
	case "fail":
		os.Stderr.WriteString("error: model not found")
		os.Exit(1)
	case "slow":
		time.Sleep(5 * time.Second)
		os.Stdout.WriteString("done")
		os.Exit(0)
	}
}

func testCommand(mode string) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cs := []string{"-test.run=TestOpenCodeHelperProcess", "--"}
		cs = append(cs, args...)
		cmd := exec.CommandContext(ctx, os.Args[0], cs...)
		cmd.Env = append(os.Environ(),
			"GO_TEST_HELPER_PROCESS=1",
			"GO_TEST_HELPER_MODE="+mode,
		)
		return cmd
	}
}

func TestOpenCodeSuccess(t *testing.T) {
	origExec := execCommand
	execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return testCommand("success")(ctx, name, args...)
	}
	defer func() { execCommand = origExec }()

	result, err := OpenCode(context.Background(), "review this", "test-model", 10*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "## Review\nScore: 8/10\nLooks good." {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestOpenCodeFailure(t *testing.T) {
	origExec := execCommand
	execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return testCommand("fail")(ctx, name, args...)
	}
	defer func() { execCommand = origExec }()

	_, err := OpenCode(context.Background(), "review this", "test-model", 10*time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestOpenCodeTimeout(t *testing.T) {
	origExec := execCommand
	execCommand = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return testCommand("slow")(ctx, name, args...)
	}
	defer func() { execCommand = origExec }()

	_, err := OpenCode(context.Background(), "review this", "test-model", 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}

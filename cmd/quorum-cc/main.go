package main

import (
	"fmt"
	"os"

	"github.com/hwuu/quorum-cc/internal/cli"
	"github.com/hwuu/quorum-cc/internal/config"
	qserver "github.com/hwuu/quorum-cc/internal/server"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "quorum-cc",
		Short: "Quorum for Claude Code — multi-model code review via OpenCode backends",
	}

	rootCmd.AddCommand(
		newServeCmd(),
		newInitCmd(),
		newStatusCmd(),
		newTestCmd(),
		newVersionCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start MCP server (stdio mode, called by Claude Code)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("load config: %w (run 'quorum-cc init' first)", err)
			}
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}
			qserver.Version = version
			return qserver.ServeStdio(cfg)
		},
	}
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Detect environment, generate config, register MCP server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Init()
		},
	}
}

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check opencode availability, configured backends, MCP registration",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Status()
		},
	}
}

func newTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test",
		Short: "Send a test review request to verify end-to-end connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cli.Test()
		},
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("quorum-cc %s\n  commit: %s\n  built:  %s\n", version, commit, date)
		},
	}
}

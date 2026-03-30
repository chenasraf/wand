package cmd

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func runShellCmd(cfg *Config, command string) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, args []string) error {
		shell := cfg.GetShell()
		cmdArgs := append([]string{"-c", command}, args...)
		cmd := exec.Command(shell, cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
}

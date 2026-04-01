package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

// stdinReader is the reader used for confirm prompts (overridable in tests).
var stdinReader io.Reader = os.Stdin

func runShellCmd(cfg *Config, command Command) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		if msg, ok := command.GetConfirmMessage(); ok {
			if !promptConfirm(msg, command.GetConfirmDefault()) {
				return nil
			}
		}

		shell := cfg.GetShell()
		cmdArgs := append([]string{"-c", command.Cmd, "_"}, args...)
		cmd := exec.Command(shell, cmdArgs...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = buildEnv(cfg, command, c)
		if command.WorkingDir != "" {
			cmd.Dir = expandPath(command.WorkingDir)
		}
		return cmd.Run()
	}
}

func promptConfirm(message string, defaultYes bool) bool {
	hint := lo.Ternary(defaultYes, "[Y/n]", "[y/N]")
	fmt.Fprintf(os.Stderr, "%s %s ", message, hint)
	scanner := bufio.NewScanner(stdinReader)
	if scanner.Scan() {
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer == "" {
			return defaultYes
		}
		return answer == "y" || answer == "yes"
	}
	return false
}

func buildEnv(cfg *Config, command Command, c *cobra.Command) []string {
	env := os.Environ()
	env = append(env, mapToEnvSlice(cfg.Env)...)
	env = append(env, mapToEnvSlice(command.Env)...)
	env = append(env, flagsToEnv(c, command.Flags)...)
	return env
}

func mapToEnvSlice(m map[string]string) []string {
	return lo.MapToSlice(m, func(k, v string) string {
		return k + "=" + v
	})
}

func flagsToEnv(c *cobra.Command, flags map[string]Flag) []string {
	return lo.MapToSlice(flags, func(name string, flag Flag) string {
		envKey := "WAND_FLAG_" + strings.ToUpper(name)
		if flag.Type == "bool" {
			val, _ := c.Flags().GetBool(name)
			return fmt.Sprintf("%s=%t", envKey, val)
		}
		val, _ := c.Flags().GetString(name)
		return fmt.Sprintf("%s=%s", envKey, val)
	})
}

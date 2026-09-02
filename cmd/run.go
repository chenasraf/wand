package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

// stdinReader is the reader used for confirm prompts (overridable in tests).
var stdinReader io.Reader = os.Stdin

// dispatchWandEntry runs another wand command, given pre-split argv (e.g. ["lint", "-v"]).
// It is wired up in Execute() once the root cobra command exists; tests may override it.
var dispatchWandEntry func(args []string) error

func runShellCmd(cfg *Config, command Command) func(*cobra.Command, []string) error {
	return func(c *cobra.Command, args []string) error {
		exp := expansion{cmd: c, args: args, global: cfg.Flags, local: command.Flags}

		if msg, ok := command.GetConfirmMessage(); ok {
			if !promptConfirm(exp.expand(msg), command.GetConfirmDefault()) {
				return nil
			}
		}

		for _, entry := range command.Pre {
			if err := runDispatchEntry(entry, exp); err != nil {
				return err
			}
		}

		if command.Cmd != "" {
			if err := execShell(cfg, command, c, args); err != nil {
				return err
			}
		}

		for _, entry := range command.Post {
			if err := runDispatchEntry(entry, exp); err != nil {
				return err
			}
		}
		return nil
	}
}

func execShell(cfg *Config, command Command, c *cobra.Command, args []string) error {
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

func runDispatchEntry(entry string, exp expansion) error {
	tokens, err := shellSplit(exp.expand(entry))
	if err != nil {
		return fmt.Errorf("invalid pre/post entry %q: %w", entry, err)
	}
	if len(tokens) == 0 {
		return nil
	}
	if dispatchWandEntry == nil {
		return fmt.Errorf("pre/post dispatcher not initialized")
	}
	return dispatchWandEntry(tokens)
}

// expansion resolves the $VAR references a command's confirm prompt and its
// pre/post entries are written against. The command's own `cmd` is expanded by
// the shell instead, so both see the same names.
type expansion struct {
	cmd    *cobra.Command
	args   []string
	global map[string]Flag
	local  map[string]Flag
}

// expand resolves $VAR and ${VAR}: positional arguments, then WAND_FLAG_<NAME>
// against the command's flags, then the process environment.
func (e expansion) expand(s string) string {
	return os.Expand(s, func(key string) string {
		if v, ok := e.positional(key); ok {
			return v
		}
		if v, ok := e.flag(key); ok {
			return v
		}
		return os.Getenv(key)
	})
}

// positional resolves $1, $2, … and $@ / $*, matching how the shell expands them
// in a command's `cmd`. An index past the end is empty, as in a shell.
func (e expansion) positional(key string) (string, bool) {
	if key == "@" || key == "*" {
		return strings.Join(e.args, " "), true
	}
	n, err := strconv.Atoi(key)
	if err != nil || n < 1 {
		return "", false
	}
	if n > len(e.args) {
		return "", true
	}
	return e.args[n-1], true
}

// flag resolves WAND_FLAG_<NAME> to a flag's current value, preferring the
// command's own flag over a global one of the same name.
func (e expansion) flag(key string) (string, bool) {
	if e.cmd == nil || !strings.HasPrefix(key, "WAND_FLAG_") {
		return "", false
	}
	for _, flags := range []map[string]Flag{e.local, e.global} {
		for name := range flags {
			if flagEnvKey(name) != key {
				continue
			}
			if f := e.cmd.Flags().Lookup(name); f != nil {
				return f.Value.String(), true
			}
		}
	}
	return "", false
}

// shellSplit performs minimal shell-style word splitting that honors single and
// double quotes and backslash escapes. Sufficient for parsing pre/post entries.
func shellSplit(s string) ([]string, error) {
	var result []string
	var current strings.Builder
	var quote byte
	inToken := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote == 0 && (c == ' ' || c == '\t' || c == '\n'):
			if inToken {
				result = append(result, current.String())
				current.Reset()
				inToken = false
			}
		case quote == 0 && (c == '"' || c == '\''):
			quote = c
			inToken = true
		case c == quote:
			quote = 0
		case quote != '\'' && c == '\\' && i+1 < len(s):
			i++
			current.WriteByte(s[i])
			inToken = true
		default:
			current.WriteByte(c)
			inToken = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unmatched %c quote", quote)
	}
	if inToken {
		result = append(result, current.String())
	}
	return result, nil
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
	// command flags last: a command flag of the same name shadows the global one
	env = append(env, flagsToEnv(c, cfg.Flags)...)
	env = append(env, flagsToEnv(c, command.Flags)...)
	return env
}

func mapToEnvSlice(m map[string]string) []string {
	return lo.MapToSlice(m, func(k, v string) string {
		return k + "=" + v
	})
}

// flagEnvKey maps a flag name to the environment variable carrying its value.
// Hyphens become underscores: a flag named "dry-run" would otherwise produce
// WAND_FLAG_DRY-RUN, which no shell can reference.
func flagEnvKey(name string) string {
	return "WAND_FLAG_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
}

func flagsToEnv(c *cobra.Command, flags map[string]Flag) []string {
	return lo.MapToSlice(flags, func(name string, flag Flag) string {
		envKey := flagEnvKey(name)
		if flag.Type == "bool" {
			val, _ := c.Flags().GetBool(name)
			return fmt.Sprintf("%s=%t", envKey, val)
		}
		val, _ := c.Flags().GetString(name)
		return fmt.Sprintf("%s=%s", envKey, val)
	})
}

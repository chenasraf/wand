package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func setArgs(args ...string) []string {
	orig := os.Args
	os.Args = args
	return orig
}

func restoreArgs(orig []string) {
	os.Args = orig
}

func TestRunShellCmd_Basic(t *testing.T) {
	cfg := &Config{Shell: "sh"}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo hello"})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "hello" {
		t.Errorf("output = %q, want hello", got)
	}
}

func TestRunShellCmd_UsesConfiguredShell(t *testing.T) {
	cfg := &Config{Shell: "/bin/sh"}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo running"})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "running" {
		t.Errorf("output = %q, want running", got)
	}
}

func TestRunShellCmd_FailingCommand(t *testing.T) {
	cfg := &Config{Shell: "sh"}

	fn := runShellCmd(cfg, Command{Cmd: "exit 1"})
	err := fn(nil, nil)

	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestRunShellCmd_PositionalArgs(t *testing.T) {
	cfg := &Config{Shell: "sh"}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo $1 $2"})
	err := fn(nil, []string{"hello", "world"})

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "hello world" {
		t.Errorf("output = %q, want \"hello world\"", got)
	}
}

func TestRunShellCmd_AllArgs(t *testing.T) {
	cfg := &Config{Shell: "sh"}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo $@"})
	err := fn(nil, []string{"a", "b", "c"})

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "a b c" {
		t.Errorf("output = %q, want \"a b c\"", got)
	}
}

func TestRunShellCmd_CommandEnv(t *testing.T) {
	cfg := &Config{Shell: "sh"}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{
		Cmd: "echo $MY_VAR",
		Env: map[string]string{"MY_VAR": "hello"},
	})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "hello" {
		t.Errorf("output = %q, want hello", got)
	}
}

func TestRunShellCmd_GlobalEnv(t *testing.T) {
	cfg := &Config{Shell: "sh", Env: map[string]string{"GLOBAL_VAR": "world"}}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo $GLOBAL_VAR"})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "world" {
		t.Errorf("output = %q, want world", got)
	}
}

func TestRunShellCmd_CommandEnvOverridesGlobal(t *testing.T) {
	cfg := &Config{Shell: "sh", Env: map[string]string{"MY_VAR": "global"}}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{
		Cmd: "echo $MY_VAR",
		Env: map[string]string{"MY_VAR": "local"},
	})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "local" {
		t.Errorf("output = %q, want local (command env should override global)", got)
	}
}

func TestRunShellCmd_WorkingDir(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	dir := t.TempDir()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "pwd", WorkingDir: dir})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	// Resolve symlinks for macOS /tmp → /private/tmp
	resolved, _ := filepath.EvalSymlinks(dir)
	if got := strings.TrimSpace(buf.String()); got != resolved {
		t.Errorf("output = %q, want %q", got, resolved)
	}
}

func TestRunShellCmd_WorkingDirTilde(t *testing.T) {
	cfg := &Config{Shell: "sh"}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "pwd", WorkingDir: "~"})
	err = fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	// Resolve symlinks for macOS /tmp → /private/tmp etc.
	resolved, _ := filepath.EvalSymlinks(home)
	if got := strings.TrimSpace(buf.String()); got != resolved {
		t.Errorf("output = %q, want %q", got, resolved)
	}
}

func TestRunShellCmd_ConfirmYes(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	origReader := stdinReader
	stdinReader = strings.NewReader("y\n")
	defer func() { stdinReader = origReader }()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo confirmed", Confirm: true})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "confirmed" {
		t.Errorf("output = %q, want confirmed", got)
	}
}

func TestRunShellCmd_ConfirmNo(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	origReader := stdinReader
	stdinReader = strings.NewReader("n\n")
	defer func() { stdinReader = origReader }()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo should-not-run", Confirm: true})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "" {
		t.Errorf("output = %q, want empty (command should not run)", got)
	}
}

func TestRunShellCmd_ConfirmCustomMessage(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	origReader := stdinReader
	stdinReader = strings.NewReader("yes\n")
	defer func() { stdinReader = origReader }()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo done", Confirm: "Really do it?"})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "done" {
		t.Errorf("output = %q, want done", got)
	}
}

func TestRunShellCmd_ConfirmDefaultYes_EmptyInput(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	origReader := stdinReader
	stdinReader = strings.NewReader("\n")
	defer func() { stdinReader = origReader }()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo ran", Confirm: true, ConfirmDefault: "yes"})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "ran" {
		t.Errorf("output = %q, want ran (default yes should run on empty input)", got)
	}
}

func TestRunShellCmd_ConfirmDefaultNo_EmptyInput(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	origReader := stdinReader
	stdinReader = strings.NewReader("\n")
	defer func() { stdinReader = origReader }()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{Cmd: "echo should-not-run", Confirm: true})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "" {
		t.Errorf("output = %q, want empty (default no should skip on empty input)", got)
	}
}

func TestRunShellCmd_InvalidShell(t *testing.T) {
	cfg := &Config{Shell: "/nonexistent/shell"}

	fn := runShellCmd(cfg, Command{Cmd: "echo hello"})
	err := fn(nil, nil)

	if err == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestShellSplit(t *testing.T) {
	cases := []struct {
		in      string
		want    []string
		wantErr bool
	}{
		{"", nil, false},
		{"   ", nil, false},
		{"one two three", []string{"one", "two", "three"}, false},
		{`build --output ./dist`, []string{"build", "--output", "./dist"}, false},
		{`echo "hello world"`, []string{"echo", "hello world"}, false},
		{`echo 'a b' c`, []string{"echo", "a b", "c"}, false},
		{`echo a\ b`, []string{"echo", "a b"}, false},
		{`echo "with \"quotes\""`, []string{"echo", `with "quotes"`}, false},
		{`unmatched "quote`, nil, true},
	}
	for _, tc := range cases {
		got, err := shellSplit(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("shellSplit(%q): expected error, got %v", tc.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("shellSplit(%q): unexpected error: %v", tc.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("shellSplit(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestRunShellCmd_PreRunsBeforeMain(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	var calls []string
	orig := dispatchWandEntry
	dispatchWandEntry = func(args []string) error {
		calls = append(calls, strings.Join(args, " "))
		return nil
	}
	defer func() { dispatchWandEntry = orig }()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{
		Cmd: "echo main",
		Pre: []string{"lint", "test --verbose"},
	})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}
	if got, want := calls, []string{"lint", "test --verbose"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pre calls = %v, want %v", got, want)
	}
	if got := strings.TrimSpace(buf.String()); got != "main" {
		t.Errorf("output = %q, want main", got)
	}
}

func TestRunShellCmd_PostRunsAfterMain(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	var calls []string
	orig := dispatchWandEntry
	dispatchWandEntry = func(args []string) error {
		calls = append(calls, strings.Join(args, " "))
		return nil
	}
	defer func() { dispatchWandEntry = orig }()

	fn := runShellCmd(cfg, Command{
		Cmd:  "true",
		Post: []string{"cleanup --force"},
	})
	if err := fn(nil, nil); err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}
	if got, want := calls, []string{"cleanup --force"}; !reflect.DeepEqual(got, want) {
		t.Errorf("post calls = %v, want %v", got, want)
	}
}

func TestRunShellCmd_PreFailureAbortsMainAndPost(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	var calls []string
	orig := dispatchWandEntry
	dispatchWandEntry = func(args []string) error {
		calls = append(calls, strings.Join(args, " "))
		if args[0] == "boom" {
			return fmt.Errorf("boom failed")
		}
		return nil
	}
	defer func() { dispatchWandEntry = orig }()

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn := runShellCmd(cfg, Command{
		Cmd:  "echo main",
		Pre:  []string{"boom", "never"},
		Post: []string{"also-never"},
	})
	err := fn(nil, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err == nil {
		t.Fatal("expected error from failing pre entry")
	}
	if got, want := calls, []string{"boom"}; !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v (subsequent pre + post must not run)", got, want)
	}
	if got := strings.TrimSpace(buf.String()); got != "" {
		t.Errorf("output = %q, want empty (main must not run)", got)
	}
}

func TestRunShellCmd_PostSkippedOnMainFailure(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	var calls []string
	orig := dispatchWandEntry
	dispatchWandEntry = func(args []string) error {
		calls = append(calls, strings.Join(args, " "))
		return nil
	}
	defer func() { dispatchWandEntry = orig }()

	fn := runShellCmd(cfg, Command{
		Cmd:  "exit 1",
		Post: []string{"never"},
	})
	if err := fn(nil, nil); err == nil {
		t.Fatal("expected error from failing main")
	}
	if len(calls) != 0 {
		t.Errorf("post calls = %v, want none (main failed)", calls)
	}
}

func TestRunShellCmd_EntryExpandsWandFlag(t *testing.T) {
	var calls [][]string
	orig := dispatchWandEntry
	dispatchWandEntry = func(args []string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	defer func() { dispatchWandEntry = orig }()

	parent := &cobra.Command{Use: "parent"}
	parent.Flags().String("target", "alice", "")
	parentCmd := Command{Flags: map[string]Flag{"target": {}}}

	exp := expansion{cmd: parent, local: parentCmd.Flags}
	if err := runDispatchEntry("greet --who $WAND_FLAG_TARGET", exp); err != nil {
		t.Fatalf("runDispatchEntry failed: %v", err)
	}
	if got, want := calls, [][]string{{"greet", "--who", "alice"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("dispatched args = %v, want %v", got, want)
	}
}

func TestRunShellCmd_NoCmdRunsOnlyPrePost(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	var calls []string
	orig := dispatchWandEntry
	dispatchWandEntry = func(args []string) error {
		calls = append(calls, args[0])
		return nil
	}
	defer func() { dispatchWandEntry = orig }()

	fn := runShellCmd(cfg, Command{
		Pre:  []string{"a"},
		Post: []string{"b"},
	})
	if err := fn(nil, nil); err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}
	if got, want := calls, []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("calls = %v, want %v", got, want)
	}
}

func TestFlagEnvKey(t *testing.T) {
	tests := map[string]string{
		"output":    "WAND_FLAG_OUTPUT",
		"dry-run":   "WAND_FLAG_DRY_RUN",
		"log-level": "WAND_FLAG_LOG_LEVEL",
		"dry_run":   "WAND_FLAG_DRY_RUN",
	}
	for name, want := range tests {
		if got := flagEnvKey(name); got != want {
			t.Errorf("flagEnvKey(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestRunShellCmd_HyphenatedFlagReachesShell(t *testing.T) {
	cfg := &Config{Shell: "sh"}
	command := Command{
		Cmd:   "echo $WAND_FLAG_DRY_RUN",
		Flags: map[string]Flag{"dry-run": {Type: "bool"}},
	}

	c := &cobra.Command{Use: "deploy"}
	registerFlags(c, command.Flags)
	if err := c.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := runShellCmd(cfg, command)(c, nil)

	_ = w.Close()
	os.Stdout = origStdout
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "true" {
		t.Errorf("output = %q, want true (hyphenated flag should map to WAND_FLAG_DRY_RUN)", got)
	}
}

func TestRunShellCmd_ConfirmExpandsVars(t *testing.T) {
	cfg := &Config{Shell: "sh", Flags: map[string]Flag{"log-level": {Default: "info"}}}
	command := Command{
		Cmd:     "true",
		Confirm: "Deploy $1 at $WAND_FLAG_LOG_LEVEL (dry=$WAND_FLAG_DRY_RUN)?",
		Flags:   map[string]Flag{"dry-run": {Type: "bool"}},
	}

	c := &cobra.Command{Use: "deploy"}
	registerFlags(c, command.Flags)
	registerFlagsOn(c.Flags(), cfg.Flags)
	if err := c.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}

	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	origStdin := stdinReader
	stdinReader = strings.NewReader("y\n")
	defer func() { stdinReader = origStdin }()

	err := runShellCmd(cfg, command)(c, []string{"prod"})

	_ = w.Close()
	os.Stderr = origStderr
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}
	want := "Deploy prod at info (dry=true)? [y/N]"
	if got := strings.TrimSpace(buf.String()); got != want {
		t.Errorf("prompt = %q, want %q", got, want)
	}
}

func TestExpansion_Positional(t *testing.T) {
	exp := expansion{args: []string{"one", "two"}}

	tests := map[string]string{
		"$1":     "one",
		"$2":     "two",
		"$3":     "",
		"${1}":   "one",
		"$@":     "one two",
		"$*":     "one two",
		"a $1 b": "a one b",
	}
	for in, want := range tests {
		if got := exp.expand(in); got != want {
			t.Errorf("expand(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestExpansion_LocalFlagShadowsGlobal(t *testing.T) {
	c := &cobra.Command{Use: "deploy"}
	c.Flags().String("profile", "local", "")

	exp := expansion{
		cmd:    c,
		local:  map[string]Flag{"profile": {}},
		global: map[string]Flag{"profile": {}},
	}

	if got := exp.expand("$WAND_FLAG_PROFILE"); got != "local" {
		t.Errorf("expand = %q, want local", got)
	}
}

func TestRunShellCmd_PreEntryExpandsPositional(t *testing.T) {
	var calls [][]string
	orig := dispatchWandEntry
	dispatchWandEntry = func(args []string) error {
		calls = append(calls, append([]string(nil), args...))
		return nil
	}
	defer func() { dispatchWandEntry = orig }()

	cfg := &Config{Shell: "sh"}
	command := Command{Pre: []string{`notify "releasing $1"`}}

	if err := runShellCmd(cfg, command)(&cobra.Command{Use: "release"}, []string{"v1.2.3"}); err != nil {
		t.Fatalf("runShellCmd failed: %v", err)
	}
	if got, want := calls, [][]string{{"notify", "releasing v1.2.3"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("dispatched args = %v, want %v", got, want)
	}
}

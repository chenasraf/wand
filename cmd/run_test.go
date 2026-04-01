package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

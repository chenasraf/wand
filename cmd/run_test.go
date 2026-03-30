package cmd

import (
	"bytes"
	"os"
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

	fn := runShellCmd(cfg, "echo hello")
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

	fn := runShellCmd(cfg, "echo running")
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

	fn := runShellCmd(cfg, "exit 1")
	err := fn(nil, nil)

	if err == nil {
		t.Error("expected error for failing command")
	}
}

func TestRunShellCmd_InvalidShell(t *testing.T) {
	cfg := &Config{Shell: "/nonexistent/shell"}

	fn := runShellCmd(cfg, "echo hello")
	err := fn(nil, nil)

	if err == nil {
		t.Error("expected error for invalid shell")
	}
}

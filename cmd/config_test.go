package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/viper"
)

func TestGetShell_String(t *testing.T) {
	cfg := &Config{Shell: "/bin/zsh"}
	if got := cfg.GetShell(); got != "/bin/zsh" {
		t.Errorf("GetShell() = %q, want /bin/zsh", got)
	}
}

func TestGetShell_PerOS(t *testing.T) {
	cfg := &Config{Shell: map[string]interface{}{
		"macos":   "/bin/zsh",
		"linux":   "/bin/bash",
		"windows": "cmd",
	}}

	got := cfg.GetShell()
	expected := map[string]string{
		"macos":   "/bin/zsh",
		"linux":   "/bin/bash",
		"windows": "cmd",
	}

	key := runtimeOS()
	if got != expected[key] {
		t.Errorf("GetShell() = %q, want %q (os=%s)", got, expected[key], key)
	}
}

func TestGetShell_FallbackToEnv(t *testing.T) {
	cfg := &Config{Shell: nil}
	t.Setenv("SHELL", "/bin/bash")

	if got := cfg.GetShell(); got != "/bin/bash" {
		t.Errorf("GetShell() = %q, want /bin/bash", got)
	}
}

func TestGetShell_FallbackToSh(t *testing.T) {
	cfg := &Config{Shell: nil}
	t.Setenv("SHELL", "")

	if got := cfg.GetShell(); got != "sh" {
		t.Errorf("GetShell() = %q, want sh", got)
	}
}

func TestGetShell_PerOS_MissingKey(t *testing.T) {
	// Map with no matching OS key should fall back
	cfg := &Config{Shell: map[string]interface{}{
		"nonexistent_os": "/bin/fake",
	}}
	t.Setenv("SHELL", "/bin/fallback")

	if got := cfg.GetShell(); got != "/bin/fallback" {
		t.Errorf("GetShell() = %q, want /bin/fallback", got)
	}
}

func TestRuntimeOS(t *testing.T) {
	got := runtimeOS()
	if runtime.GOOS == "darwin" {
		if got != "macos" {
			t.Errorf("runtimeOS() = %q, want macos", got)
		}
	} else {
		if got != runtime.GOOS {
			t.Errorf("runtimeOS() = %q, want %q", got, runtime.GOOS)
		}
	}
}

func writeTestConfig(t *testing.T, dir, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, "wand.yml"), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func setupTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	writeTestConfig(t, dir, content)

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		viper.Reset()
	})

	return dir
}

func TestLoadConfig_Basic(t *testing.T) {
	setupTestConfig(t, `
main:
  description: test main
  cmd: echo hello

build:
  description: build it
  cmd: go build
`)

	cfg, commands, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if cfg == nil {
		t.Fatal("cfg is nil")
	}

	if len(commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(commands))
	}

	if commands["main"].Description != "test main" {
		t.Errorf("main description = %q", commands["main"].Description)
	}
	if commands["main"].Cmd != "echo hello" {
		t.Errorf("main cmd = %q", commands["main"].Cmd)
	}
	if commands["build"].Description != "build it" {
		t.Errorf("build description = %q", commands["build"].Description)
	}
}

func TestLoadConfig_WithChildren(t *testing.T) {
	setupTestConfig(t, `
parent:
  description: parent cmd
  cmd: echo parent
  children:
    child:
      description: child cmd
      cmd: echo child
      children:
        grandchild:
          description: grandchild cmd
          cmd: echo grandchild
`)

	_, commands, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	parent := commands["parent"]
	if len(parent.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(parent.Children))
	}

	child := parent.Children["child"]
	if child.Description != "child cmd" {
		t.Errorf("child description = %q", child.Description)
	}

	grandchild := child.Children["grandchild"]
	if grandchild.Cmd != "echo grandchild" {
		t.Errorf("grandchild cmd = %q", grandchild.Cmd)
	}
}

func TestLoadConfig_WithShellString(t *testing.T) {
	setupTestConfig(t, `
.config:
  shell: /bin/zsh

main:
  description: test
  cmd: echo test
`)

	cfg, commands, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.GetShell() != "/bin/zsh" {
		t.Errorf("shell = %q, want /bin/zsh", cfg.GetShell())
	}

	if _, ok := commands[".config"]; ok {
		t.Error(".config should not appear in commands")
	}
}

func TestLoadConfig_WithShellPerOS(t *testing.T) {
	setupTestConfig(t, `
.config:
  shell:
    macos: /bin/zsh
    linux: /bin/bash
    windows: cmd

main:
  description: test
  cmd: echo test
`)

	cfg, _, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	key := runtimeOS()
	expected := map[string]string{
		"macos":   "/bin/zsh",
		"linux":   "/bin/bash",
		"windows": "cmd",
	}
	if cfg.GetShell() != expected[key] {
		t.Errorf("shell = %q, want %q", cfg.GetShell(), expected[key])
	}
}

func TestLoadConfig_WithGlobalEnv(t *testing.T) {
	setupTestConfig(t, `
.config:
  env:
    FOO: bar

main:
  description: test
  cmd: echo test
`)

	cfg, _, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Env["FOO"] != "bar" {
		t.Errorf("global env FOO = %q, want bar", cfg.Env["FOO"])
	}
}

func TestLoadConfig_WithCommandEnv(t *testing.T) {
	setupTestConfig(t, `
main:
  description: test
  cmd: echo test
  env:
    MY_VAR: hello
`)

	_, commands, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if commands["main"].Env["MY_VAR"] != "hello" {
		t.Errorf("command env MY_VAR = %q, want hello", commands["main"].Env["MY_VAR"])
	}
}

func TestLoadConfig_NoConfigFile(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	_ = os.Chdir(dir)
	defer func() {
		_ = os.Chdir(origDir)
		viper.Reset()
	}()

	// Remove HOME to prevent finding a real ~/.wand.yml
	t.Setenv("HOME", dir)

	_, _, err := loadConfig()
	if err == nil {
		t.Error("expected error for missing config, got nil")
	}
}

func TestLoadConfig_SearchUpward(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}

	writeTestConfig(t, parent, `
main:
  description: found it
  cmd: echo found
`)

	origDir, _ := os.Getwd()
	_ = os.Chdir(child)
	defer func() {
		_ = os.Chdir(origDir)
		viper.Reset()
	}()

	_, commands, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}

	if commands["main"].Description != "found it" {
		t.Errorf("expected to find config in parent dir, got description=%q", commands["main"].Description)
	}
}

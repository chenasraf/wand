package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestBuildCobraCommand_Basic(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Description: "test command",
		Cmd:         "echo test",
	}

	c := buildCobraCommand(cfg, "mycommand", cmd)

	if c.Use != "mycommand" {
		t.Errorf("Use = %q, want mycommand", c.Use)
	}
	if c.Short != "test command" {
		t.Errorf("Short = %q, want 'test command'", c.Short)
	}
	if c.RunE == nil {
		t.Error("RunE should be set when Cmd is non-empty")
	}
	if c.Hidden {
		t.Error("command without underscore prefix should not be hidden")
	}
}

func TestBuildCobraCommand_UnderscorePrefixIsHidden(t *testing.T) {
	cfg := &Config{}
	cmd := Command{Description: "private", Cmd: "echo private"}

	c := buildCobraCommand(cfg, "_setup", cmd)

	if !c.Hidden {
		t.Error("command with _ prefix should be Hidden")
	}
	if c.RunE == nil {
		t.Error("hidden command should still be runnable")
	}
}

func TestBuildCobraCommand_UnderscorePrefixChildHidden(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Cmd: "echo parent",
		Children: map[string]Command{
			"_internal": {Cmd: "echo internal"},
			"public":    {Cmd: "echo public"},
		},
	}

	c := buildCobraCommand(cfg, "parent", cmd)

	for _, child := range c.Commands() {
		switch child.Use {
		case "_internal":
			if !child.Hidden {
				t.Error("_internal child should be Hidden")
			}
		case "public":
			if child.Hidden {
				t.Error("public child should not be Hidden")
			}
		}
	}
}

func TestBuildCobraCommand_NoCmd(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Description: "parent only",
		Children: map[string]Command{
			"child": {Description: "child cmd", Cmd: "echo child"},
		},
	}

	c := buildCobraCommand(cfg, "parent", cmd)

	if c.RunE != nil {
		t.Error("RunE should be nil when Cmd is empty")
	}
	if !c.HasSubCommands() {
		t.Error("expected subcommands")
	}
}

func TestBuildCobraCommand_Children(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Description: "parent",
		Cmd:         "echo parent",
		Children: map[string]Command{
			"child1": {Description: "first child", Cmd: "echo child1"},
			"child2": {
				Description: "second child",
				Cmd:         "echo child2",
				Children: map[string]Command{
					"grandchild": {Description: "grandchild", Cmd: "echo grandchild"},
				},
			},
		},
	}

	c := buildCobraCommand(cfg, "parent", cmd)

	children := c.Commands()
	if len(children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(children))
	}

	// Find child2 and check its grandchild
	var child2Found bool
	for _, child := range children {
		if child.Use == "child2" {
			child2Found = true
			grandchildren := child.Commands()
			if len(grandchildren) != 1 {
				t.Fatalf("expected 1 grandchild, got %d", len(grandchildren))
			}
			if grandchildren[0].Use != "grandchild" {
				t.Errorf("grandchild Use = %q", grandchildren[0].Use)
			}
		}
	}
	if !child2Found {
		t.Error("child2 not found in subcommands")
	}
}

func TestExecute_WithConfig(t *testing.T) {
	setupTestConfig(t, `
main:
  description: test main
  cmd: echo hello
`)

	// Execute should succeed - the root command runs "echo hello"
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestExecute_Subcommand(t *testing.T) {
	setupTestConfig(t, `
greet:
  description: say hi
  cmd: echo hi
`)

	// Simulate running "wand greet" by setting os.Args
	origArgs := setArgs("wand", "greet")
	defer restoreArgs(origArgs)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestExecute_NestedSubcommand(t *testing.T) {
	setupTestConfig(t, `
parent:
  description: parent
  cmd: echo parent
  children:
    child:
      description: child
      cmd: echo child
`)

	origArgs := setArgs("wand", "parent", "child")
	defer restoreArgs(origArgs)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestBuildCobraCommand_WithStringFlags(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Description: "flagged",
		Cmd:         "echo $WAND_FLAG_OUTPUT",
		Flags: map[string]Flag{
			"output": {Alias: "o", Description: "output path", Default: "default.txt"},
		},
	}

	c := buildCobraCommand(cfg, "build", cmd)

	f := c.Flags().Lookup("output")
	if f == nil {
		t.Fatal("flag 'output' not registered")
	}
	if f.Shorthand != "o" {
		t.Errorf("shorthand = %q, want o", f.Shorthand)
	}
	if f.Usage != "output path" {
		t.Errorf("usage = %q", f.Usage)
	}
	if f.DefValue != "default.txt" {
		t.Errorf("default = %q, want default.txt", f.DefValue)
	}
}

func TestBuildCobraCommand_WithBoolFlags(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Description: "verbose",
		Cmd:         "echo $WAND_FLAG_VERBOSE",
		Flags: map[string]Flag{
			"verbose": {Alias: "v", Description: "verbose output", Type: "bool"},
		},
	}

	c := buildCobraCommand(cfg, "run", cmd)

	f := c.Flags().Lookup("verbose")
	if f == nil {
		t.Fatal("flag 'verbose' not registered")
	}
	if f.Shorthand != "v" {
		t.Errorf("shorthand = %q, want v", f.Shorthand)
	}
	if f.DefValue != "false" {
		t.Errorf("default = %q, want false", f.DefValue)
	}
}

func TestBuildCobraCommand_FlagNoAlias(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Description: "test",
		Cmd:         "echo test",
		Flags: map[string]Flag{
			"count": {Description: "a count"},
		},
	}

	c := buildCobraCommand(cfg, "test", cmd)

	f := c.Flags().Lookup("count")
	if f == nil {
		t.Fatal("flag 'count' not registered")
	}
	if f.Shorthand != "" {
		t.Errorf("shorthand = %q, want empty", f.Shorthand)
	}
}

func TestExecute_WithStringFlag(t *testing.T) {
	setupTestConfig(t, `
greet:
  description: greet
  cmd: echo $WAND_FLAG_NAME
  flags:
    name:
      alias: n
      description: who to greet
`)

	origArgs := setArgs("wand", "greet", "--name", "world")
	defer restoreArgs(origArgs)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestExecute_WithBoolFlag(t *testing.T) {
	setupTestConfig(t, `
greet:
  description: greet
  cmd: echo "verbose=$WAND_FLAG_VERBOSE"
  flags:
    verbose:
      alias: v
      description: verbose output
      type: bool
`)

	origArgs := setArgs("wand", "greet", "--verbose")
	defer restoreArgs(origArgs)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestBuildCobraCommand_WithAliases(t *testing.T) {
	cfg := &Config{}
	cmd := Command{
		Description: "build",
		Cmd:         "echo build",
		Aliases:     []string{"b", "compile"},
	}

	c := buildCobraCommand(cfg, "build", cmd)

	if len(c.Aliases) != 2 {
		t.Fatalf("expected 2 aliases, got %d", len(c.Aliases))
	}
	if c.Aliases[0] != "b" || c.Aliases[1] != "compile" {
		t.Errorf("aliases = %v, want [b compile]", c.Aliases)
	}
}

func TestExecute_WithAlias(t *testing.T) {
	setupTestConfig(t, `
build:
  description: build
  cmd: echo built
  aliases: [b]
`)

	origArgs := setArgs("wand", "b")
	defer restoreArgs(origArgs)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestResolveConfigFile_Flag(t *testing.T) {
	origArgs := setArgs("wand", "--wand-file", "/tmp/custom.yml", "build")
	defer restoreArgs(origArgs)

	got := resolveConfigFile()
	if got != "/tmp/custom.yml" {
		t.Errorf("resolveConfigFile() = %q, want /tmp/custom.yml", got)
	}
}

func TestResolveConfigFile_FlagEquals(t *testing.T) {
	origArgs := setArgs("wand", "--wand-file=/tmp/custom.yml", "build")
	defer restoreArgs(origArgs)

	got := resolveConfigFile()
	if got != "/tmp/custom.yml" {
		t.Errorf("resolveConfigFile() = %q, want /tmp/custom.yml", got)
	}
}

func TestResolveConfigFile_EnvVar(t *testing.T) {
	origArgs := setArgs("wand", "build")
	defer restoreArgs(origArgs)

	t.Setenv("WAND_FILE", "/tmp/env.yml")

	got := resolveConfigFile()
	if got != "/tmp/env.yml" {
		t.Errorf("resolveConfigFile() = %q, want /tmp/env.yml", got)
	}
}

func TestResolveConfigFile_FlagOverridesEnv(t *testing.T) {
	origArgs := setArgs("wand", "--wand-file", "/tmp/flag.yml")
	defer restoreArgs(origArgs)

	t.Setenv("WAND_FILE", "/tmp/env.yml")

	got := resolveConfigFile()
	if got != "/tmp/flag.yml" {
		t.Errorf("resolveConfigFile() = %q, want /tmp/flag.yml (flag should override env)", got)
	}
}

func TestExecute_WithWandFileFlag(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom.yml")
	_ = os.WriteFile(configPath, []byte(`
main:
  description: custom
  cmd: echo custom
`), 0644)

	origArgs := setArgs("wand", "--wand-file", configPath)
	defer restoreArgs(origArgs)

	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestExecute_NoMain(t *testing.T) {
	setupTestConfig(t, `
build:
  description: build
  cmd: echo build
`)

	// Running with no args and no "main" should print help (no error)
	err := Execute()
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestExecute_GlobalFlagAfterCommandName(t *testing.T) {
	setupTestConfig(t, `
.config:
  flags:
    profile:
      alias: p
      default: dev

build:
  cmd: echo $WAND_FLAG_PROFILE
`)

	origArgs := setArgs("wand", "build", "--profile", "prod")
	defer restoreArgs(origArgs)

	if got := captureStdout(t, Execute); got != "prod" {
		t.Errorf("output = %q, want prod", got)
	}
}

func TestExecute_GlobalFlagBeforeCommandName(t *testing.T) {
	setupTestConfig(t, `
.config:
  flags:
    profile:
      alias: p
      default: dev
    verbose:
      alias: V
      type: bool

build:
  cmd: echo $WAND_FLAG_PROFILE
  children:
    docs:
      cmd: echo $WAND_FLAG_PROFILE
`)

	origArgs := setArgs("wand", "-p", "prod", "--verbose", "build", "docs")
	defer restoreArgs(origArgs)

	if got := captureStdout(t, Execute); got != "prod" {
		t.Errorf("output = %q, want prod", got)
	}
}

// captureStdout runs fn with os.Stdout redirected and returns the trimmed output.
func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	runErr := fn()

	_ = w.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatalf("Execute() failed: %v", runErr)
	}
	return strings.TrimSpace(buf.String())
}

func TestExecute_InvalidGlobalFlagReturnsError(t *testing.T) {
	setupTestConfig(t, `
.config:
  flags:
    wand-file:
      description: nope

main:
  cmd: echo hello
`)

	err := Execute()
	if err == nil {
		t.Fatal("Execute() = nil, want an error for a reserved global flag name")
	}
}

func TestRegisterFlagsOn_GlobalFlags(t *testing.T) {
	c := &cobra.Command{Use: "wand"}
	registerFlagsOn(c.PersistentFlags(), map[string]Flag{
		"profile": {Alias: "p", Default: "dev", Description: "config profile"},
		"verbose": {Alias: "V", Type: "bool"},
	})

	profile := c.PersistentFlags().Lookup("profile")
	if profile == nil {
		t.Fatal("profile should be registered as a persistent flag")
	}
	if profile.Shorthand != "p" || profile.DefValue != "dev" {
		t.Errorf("profile = %+v, want shorthand p and default dev", profile)
	}
	if verbose := c.PersistentFlags().Lookup("verbose"); verbose == nil || verbose.Value.Type() != "bool" {
		t.Errorf("verbose = %+v, want a bool flag", verbose)
	}
}

func TestExecute_BinNameInHelp(t *testing.T) {
	setupTestConfig(t, `
.config:
  bin_name: nxc

build:
  description: build it
  cmd: echo build
`)

	origArgs := setArgs("wand", "--help")
	defer restoreArgs(origArgs)

	out := captureStdout(t, Execute)

	for _, want := range []string{"nxc [command]", `Use "nxc [command] --help"`} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q:\n%s", want, out)
		}
	}
	// --wand-file is wand's own flag and keeps its name
	if strings.Contains(out, "wand [command]") {
		t.Errorf("help output should not use the default name:\n%s", out)
	}
}

func TestExecute_BinNameInSubcommandHelp(t *testing.T) {
	setupTestConfig(t, `
.config:
  bin_name: nxc

build:
  description: build it
  cmd: echo build
  children:
    docs:
      description: build docs
      cmd: echo docs
`)

	origArgs := setArgs("wand", "build", "--help")
	defer restoreArgs(origArgs)

	out := captureStdout(t, Execute)

	if !strings.Contains(out, "nxc build [command]") {
		t.Errorf("subcommand help should use the configured bin name:\n%s", out)
	}
}

func TestExecute_DefaultBinName(t *testing.T) {
	setupTestConfig(t, `
build:
  description: build it
  cmd: echo build
`)

	origArgs := setArgs("wand", "--help")
	defer restoreArgs(origArgs)

	out := captureStdout(t, Execute)

	if !strings.Contains(out, "wand [command]") {
		t.Errorf("help output should default to wand:\n%s", out)
	}
}

func TestExecute_BinNameHidesWandFile(t *testing.T) {
	setupTestConfig(t, `
.config:
  bin_name: nxc

build:
  description: build it
  cmd: echo build
`)

	origArgs := setArgs("wand", "--help")
	defer restoreArgs(origArgs)

	if out := captureStdout(t, Execute); strings.Contains(out, "--wand-file") {
		t.Errorf("--wand-file should be hidden when bin_name is set:\n%s", out)
	}
}

func TestExecute_WandFileStillWorksWithBinName(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "custom.yml")
	if err := os.WriteFile(configPath, []byte(`
.config:
  bin_name: nxc

main:
  cmd: echo from-custom
`), 0644); err != nil {
		t.Fatal(err)
	}

	origArgs := setArgs("wand", "--wand-file", configPath)
	defer restoreArgs(origArgs)

	if got := captureStdout(t, Execute); got != "from-custom" {
		t.Errorf("output = %q, want from-custom (hidden flag must still parse)", got)
	}
}

func TestExecute_DefaultBinNameShowsWandFile(t *testing.T) {
	setupTestConfig(t, `
build:
  description: build it
  cmd: echo build
`)

	origArgs := setArgs("wand", "--help")
	defer restoreArgs(origArgs)

	if out := captureStdout(t, Execute); !strings.Contains(out, "--wand-file") {
		t.Errorf("--wand-file should be listed by default:\n%s", out)
	}
}

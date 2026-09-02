package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"strings"

	"github.com/samber/lo"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

type Flag struct {
	Alias       string      `yaml:"alias"`
	Description string      `yaml:"description"`
	Default     interface{} `yaml:"default"`
	Type        string      `yaml:"type"`
}

type Command struct {
	Description    string             `yaml:"description"`
	Cmd            string             `yaml:"cmd"`
	Children       map[string]Command `yaml:"children"`
	Flags          map[string]Flag    `yaml:"flags"`
	Env            map[string]string  `yaml:"env"`
	WorkingDir     string             `yaml:"working_dir"`
	Aliases        []string           `yaml:"aliases"`
	Confirm        interface{}        `yaml:"confirm"`
	ConfirmDefault string             `yaml:"confirm_default"`
	Pre            []string           `yaml:"pre"`
	Post           []string           `yaml:"post"`
}

// GetConfirmMessage returns the confirm prompt message and whether confirmation is enabled.
func (c Command) GetConfirmMessage() (string, bool) {
	switch v := c.Confirm.(type) {
	case bool:
		if v {
			return "Are you sure?", true
		}
		return "", false
	case string:
		return v, true
	}
	return "", false
}

// GetConfirmDefault returns true if the default answer is yes.
func (c Command) GetConfirmDefault() bool {
	return strings.EqualFold(c.ConfirmDefault, "yes") || strings.EqualFold(c.ConfirmDefault, "y")
}

type Config struct {
	Shell interface{}       `yaml:"shell"`
	Env   map[string]string `yaml:"env"`
	Flags map[string]Flag   `yaml:"flags"`
}

func (c *Config) GetShell() string {
	switch v := c.Shell.(type) {
	case string:
		return v
	case map[string]interface{}:
		key := runtimeOS()
		if shell, ok := v[key]; ok {
			if s, ok := shell.(string); ok {
				return s
			}
		}
	}

	// fall back to $SHELL or sh
	return lo.CoalesceOrEmpty(os.Getenv("SHELL"), "sh")
}

func runtimeOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

type rawConfig struct {
	DotConfig *Config            `yaml:".config"`
	Commands  map[string]Command `yaml:",inline"`
}

// expandPath expands a leading ~ to the user's home directory.
func expandPath(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[1:])
	}
	return path
}

func loadConfig(explicitPath string) (*Config, map[string]Command, error) {
	var configPath string
	var err error
	if explicitPath != "" {
		configPath = expandPath(explicitPath)
	} else {
		configPath, err = findConfigFile()
		if err != nil {
			return nil, nil, err
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read config: %w", err)
	}

	var raw rawConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	cfg := lo.FromPtrOr(raw.DotConfig, Config{})
	commands := lo.OmitByKeys(raw.Commands, []string{".config"})

	return &cfg, commands, nil
}

// wand's own root flags. A config flag reusing one of these names or shorthands
// collides when cobra merges the persistent flag set into a command.
var (
	reservedFlagNames   = []string{"wand-file", "help"}
	reservedFlagAliases = []string{"h"}
)

// validateFlags rejects flag definitions that cannot be registered: multi-character
// aliases, wand's own flags, and aliases claimed twice in the same flag set.
func validateFlags(scope string, flags map[string]Flag, inherited map[string]Flag) error {
	byAlias := map[string]string{}
	for name, flag := range inherited {
		if flag.Alias != "" {
			byAlias[flag.Alias] = name
		}
	}

	names := lo.Keys(flags)
	sort.Strings(names)
	for _, name := range names {
		flag := flags[name]
		if lo.Contains(reservedFlagNames, name) {
			return fmt.Errorf("%s: flag %q is reserved by wand", scope, name)
		}
		if flag.Alias == "" {
			continue
		}
		if len(flag.Alias) != 1 {
			return fmt.Errorf("%s: alias %q of flag %q must be a single character", scope, flag.Alias, name)
		}
		if lo.Contains(reservedFlagAliases, flag.Alias) {
			return fmt.Errorf("%s: alias %q of flag %q is reserved by wand", scope, flag.Alias, name)
		}
		// a command flag may reuse a global flag's name and alias to override it
		if other, ok := byAlias[flag.Alias]; ok && other != name {
			return fmt.Errorf("%s: flags %q and %q share the alias %q", scope, other, name, flag.Alias)
		}
		byAlias[flag.Alias] = name
	}
	return nil
}

// validateConfigFlags checks the global flags and every command's flags against
// the flags they will be merged with.
func validateConfigFlags(cfg *Config, commands map[string]Command) error {
	if err := validateFlags("global flags", cfg.Flags, nil); err != nil {
		return err
	}
	names := lo.Keys(commands)
	sort.Strings(names)
	for _, name := range names {
		if err := validateCommandFlags(name, commands[name], cfg.Flags); err != nil {
			return err
		}
	}
	return nil
}

func validateCommandFlags(path string, command Command, globals map[string]Flag) error {
	if err := validateFlags("command "+path, command.Flags, globals); err != nil {
		return err
	}
	names := lo.Keys(command.Children)
	sort.Strings(names)
	for _, name := range names {
		if err := validateCommandFlags(path+" "+name, command.Children[name], globals); err != nil {
			return err
		}
	}
	return nil
}

func findConfigFile() (string, error) {
	viper.SetConfigName("wand")
	viper.SetConfigType("yaml")

	// ./wand.yml
	viper.AddConfigPath(".")

	// search up from cwd
	dir, err := os.Getwd()
	if err == nil {
		for {
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
			viper.AddConfigPath(dir)
		}
	}

	// ~/.wand.yml and ~/.config/wand.yml
	home, err := os.UserHomeDir()
	if err == nil {
		viper.AddConfigPath(home)
		viper.AddConfigPath(filepath.Join(home, ".config"))
	}

	if err := viper.ReadInConfig(); err != nil {
		return "", fmt.Errorf("config file not found: %w", err)
	}

	return viper.ConfigFileUsed(), nil
}

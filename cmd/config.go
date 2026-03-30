package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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

func loadConfig(explicitPath string) (*Config, map[string]Command, error) {
	var configPath string
	var err error
	if explicitPath != "" {
		configPath = explicitPath
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

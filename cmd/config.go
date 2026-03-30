package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/samber/lo"
	"github.com/spf13/viper"
)

type Flag struct {
	Alias       string      `mapstructure:"alias"`
	Description string      `mapstructure:"description"`
	Default     interface{} `mapstructure:"default"`
	Type        string      `mapstructure:"type"`
}

type Command struct {
	Description string             `mapstructure:"description"`
	Cmd         string             `mapstructure:"cmd"`
	Children    map[string]Command `mapstructure:"children"`
	Flags       map[string]Flag    `mapstructure:"flags"`
}

type Config struct {
	Shell interface{} `mapstructure:"shell"`
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

func loadConfig() (*Config, map[string]Command, error) {
	if err := initConfigPaths(); err != nil {
		return nil, nil, err
	}

	var cfg Config
	if sub := viper.Sub(".config"); sub != nil {
		if err := sub.Unmarshal(&cfg); err != nil {
			return nil, nil, fmt.Errorf("failed to parse .config: %w", err)
		}
	}

	allEntries := make(map[string]Command)
	if err := viper.Unmarshal(&allEntries); err != nil {
		return nil, nil, fmt.Errorf("failed to parse config: %w", err)
	}

	commands := lo.OmitByKeys(allEntries, []string{".config"})
	return &cfg, commands, nil
}

func initConfigPaths() error {
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
		return fmt.Errorf("config file not found: %w", err)
	}

	return nil
}

package cmd

import (
	"os"
	"strings"

	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func Execute() error {
	configFile := resolveConfigFile()
	cfg, commands, err := loadConfig(configFile)
	if err != nil {
		return err
	}

	rootCmd := &cobra.Command{
		Use:           "wand",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().String("wand-file", "", "path to wand config file (overrides discovery)")

	if main, ok := commands["main"]; ok {
		rootCmd.Short = main.Description
		rootCmd.Args = cobra.ArbitraryArgs
		rootCmd.RunE = runShellCmd(cfg, main)
		registerFlags(rootCmd, main.Flags)
	}

	subcommands := lo.OmitByKeys(commands, []string{"main"})
	lo.ForEach(
		lo.MapToSlice(subcommands, func(name string, cmd Command) *cobra.Command {
			return buildCobraCommand(cfg, name, cmd)
		}),
		func(c *cobra.Command, _ int) {
			rootCmd.AddCommand(c)
		},
	)

	return rootCmd.Execute()
}

func buildCobraCommand(cfg *Config, name string, cmd Command) *cobra.Command {
	c := &cobra.Command{
		Use:     name,
		Aliases: cmd.Aliases,
		Short:   cmd.Description,
	}

	if cmd.Cmd != "" {
		c.Args = cobra.ArbitraryArgs
		c.RunE = runShellCmd(cfg, cmd)
		registerFlags(c, cmd.Flags)
	}

	lo.ForEach(
		lo.MapToSlice(cmd.Children, func(childName string, child Command) *cobra.Command {
			return buildCobraCommand(cfg, childName, child)
		}),
		func(child *cobra.Command, _ int) {
			c.AddCommand(child)
		},
	)

	return c
}

// resolveConfigFile extracts --wand-file from os.Args or WAND_FILE env before cobra parses.
func resolveConfigFile() string {
	for i, arg := range os.Args {
		if arg == "--wand-file" && i+1 < len(os.Args) {
			return os.Args[i+1]
		}
		if strings.HasPrefix(arg, "--wand-file=") {
			return strings.TrimPrefix(arg, "--wand-file=")
		}
	}
	return os.Getenv("WAND_FILE")
}

func registerFlags(c *cobra.Command, flags map[string]Flag) {
	lo.ForEach(
		lo.Entries(flags),
		func(e lo.Entry[string, Flag], _ int) {
			name, flag := e.Key, e.Value
			if flag.Type == "bool" {
				def, _ := flag.Default.(bool)
				if flag.Alias != "" {
					c.Flags().BoolP(name, flag.Alias, def, flag.Description)
				} else {
					c.Flags().Bool(name, def, flag.Description)
				}
			} else {
				def, _ := flag.Default.(string)
				if flag.Alias != "" {
					c.Flags().StringP(name, flag.Alias, def, flag.Description)
				} else {
					c.Flags().String(name, def, flag.Description)
				}
			}
		},
	)
}

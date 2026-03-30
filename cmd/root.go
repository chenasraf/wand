package cmd

import (
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

func Execute() error {
	cfg, commands, err := loadConfig()
	if err != nil {
		return err
	}

	rootCmd := &cobra.Command{
		Use:           "wand",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	if main, ok := commands["main"]; ok {
		rootCmd.Short = main.Description
		rootCmd.RunE = runShellCmd(cfg, main.Cmd)
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
		Use:   name,
		Short: cmd.Description,
	}

	if cmd.Cmd != "" {
		c.RunE = runShellCmd(cfg, cmd.Cmd)
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

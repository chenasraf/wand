package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// completionLong builds the completion help, showing --wand-file only when it is
// part of the tool's own interface.
func completionLong(rootCmd *cobra.Command) string {
	bin := rootCmd.Name()

	configFlag := " --wand-file ~/.config/wand/nextcloud.yml"
	if f := rootCmd.PersistentFlags().Lookup("wand-file"); f != nil && f.Hidden {
		configFlag = ""
	}

	return fmt.Sprintf(`Generate a shell completion script for %[1]s.
Use --name to generate completions for an alias:

  # Source completions for an alias directly
  source <(%[1]s%[2]s completion --name %[1]s-alias zsh)

  # Or write to a file
  %[1]s%[2]s completion --name %[1]s-alias zsh > _%[1]s-alias`, bin, configFlag)
}

func newCompletionCmd(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 "Generate shell completion script",
		Long:                  completionLong(rootCmd),
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			if name != "" {
				rootCmd.Use = name
			}

			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(os.Stdout)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}

	cmd.Flags().String("name", "", "command name to use in the completion script (for aliases)")

	return cmd
}

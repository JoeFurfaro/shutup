package cmd

import (
	"fmt"

	"github.com/joe/shutup/internal/project"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var listEnv string

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List this project's consumed variables and their state in an env",
	Long: `Lists every variable the current project consumes, with its state in the target
env (default: the project's default_env).

Public values are shown in clear. Secret values are NEVER shown — only whether
they are set.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := project.Open()
		if err != nil {
			return err
		}
		status, err := p.Status(listEnv)
		if err != nil {
			return err
		}
		if len(status) == 0 {
			ui.Info("This project consumes no variables yet. Add one with `shutup set <NAME>`.")
			return nil
		}
		for _, vs := range status {
			switch {
			case vs.Public && vs.Set:
				fmt.Printf("%s=%s\n", ui.Bold(vs.Name), ui.Value(vs.Value))
			case vs.Public && !vs.Set:
				fmt.Printf("%s %s\n", ui.Bold(vs.Name), ui.Dim("(public, not set)"))
			case !vs.Public && vs.Set:
				fmt.Printf("%s %s\n", ui.Bold(vs.Name), ui.Dim("(secret, set)"))
			default:
				fmt.Printf("%s %s\n", ui.Bold(vs.Name), ui.Dim("(not set)"))
			}
		}
		return nil
	},
}

func init() {
	addEnvFlag(listCmd, &listEnv)
	rootCmd.AddCommand(listCmd)
}

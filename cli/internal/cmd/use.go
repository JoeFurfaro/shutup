package cmd

import (
	"github.com/joe/shutup/internal/project"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use <NAME>",
	Short: "Declare that this project consumes a variable (no value)",
	Long: `Adds a variable NAME to the current project's consumed set, without setting a
value. Use this to consume a variable whose value already lives in a shared env
(e.g. one a teammate set), or to declare the contract before a value exists.

To also set the value, use ` + "`shutup set`" + ` instead. To see what's still unset, run
` + "`shutup missing`" + `.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := project.Open()
		if err != nil {
			return err
		}
		added, err := p.Use(args[0])
		if err != nil {
			return err
		}
		if added {
			ui.Success("This project now consumes %s", ui.Bold(args[0]))
		} else {
			ui.Info("%s is already consumed by this project", ui.Bold(args[0]))
		}
		return nil
	},
}

var unuseCmd = &cobra.Command{
	Use:   "unuse <NAME>",
	Short: "Stop consuming a variable in this project",
	Long: `Removes a variable NAME from the current project's consumed set. Does not touch
the env or its value — other projects consuming the same var are unaffected.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := project.Open()
		if err != nil {
			return err
		}
		removed, err := p.Unuse(args[0])
		if err != nil {
			return err
		}
		if removed {
			ui.Success("This project no longer consumes %s", ui.Bold(args[0]))
		} else {
			ui.Info("%s was not consumed by this project", ui.Bold(args[0]))
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(useCmd)
	rootCmd.AddCommand(unuseCmd)
}

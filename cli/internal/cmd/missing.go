package cmd

import (
	"fmt"

	"github.com/joe/shutup/internal/project"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var missingEnv string

var missingCmd = &cobra.Command{
	Use:   "missing",
	Short: "Show consumed variables that have no value in an env yet",
	Long: `Lists variables the current project consumes but that have no value in the target
env (default: the project's default_env).

Use this to drive setup: run it, then ` + "`shutup set <NAME>`" + ` each one. Variable names
are printed to stdout (one per line) so agents and scripts can parse them.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := project.Open()
		if err != nil {
			return err
		}
		missing, err := p.Missing(missingEnv)
		if err != nil {
			return err
		}
		envLabel := missingEnv
		if envLabel == "" {
			envLabel = p.Config.DefaultEnv
		}
		if len(missing) == 0 {
			ui.Success("All consumed variables are set for env %s", ui.Value(envLabel))
			return nil
		}
		for _, vs := range missing {
			fmt.Println(vs.Name)
		}
		ui.Info("")
		for _, vs := range missing {
			ui.Hint("`shutup set %s`  (you'll be prompted; the value stays hidden)", vs.Name)
		}
		return nil
	},
}

func init() {
	addEnvFlag(missingCmd, &missingEnv)
	rootCmd.AddCommand(missingCmd)
}

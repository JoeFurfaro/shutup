package cmd

import (
	"fmt"
	"os"

	"github.com/joe/shutup/internal/project"
	"github.com/spf13/cobra"
)

var checkEnv string

var checkCmd = &cobra.Command{
	Use:   "check <NAME>",
	Short: "Check whether a variable has a value in an env (never prints it)",
	Long: `Reports whether a variable has a value in the target env (default: the project's
default_env).

Prints "exists" (exit 0) or "not found" (exit 1) and NOTHING else — never the
value, even redacted. This is the agent-safe way to test for a secret.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := project.Open()
		if err != nil {
			return err
		}
		exists, err := p.Exists(checkEnv, args[0])
		if err != nil {
			return err
		}
		if exists {
			fmt.Println("exists")
			return nil
		}
		fmt.Println("not found")
		os.Exit(1)
		return nil
	},
}

func init() {
	addEnvFlag(checkCmd, &checkEnv)
	rootCmd.AddCommand(checkCmd)
}

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "shutup",
	Short: "Local-first env var + secrets manager, safe to use alongside AI agents",
	Long: `shutup manages environment variables and secrets, safely alongside AI agents.

MENTAL MODEL
  • An ENVIRONMENT ("env", e.g. dev/prod) is a bag of variables — each with a value
    and a secret/public flag. Envs hold all the values and live on your machine
    (~/.shutup/envs/), not in your repo.
  • A PROJECT (the dir you're in, found by walking up to shutup.config.yaml) just
    declares the variable NAMES it consumes and which envs it uses. It holds no
    values — it's safe to commit.
  • You work from inside a project. ` + "`--env`" + ` picks which env (default: the project's
    default_env). Commands resolve the project's consumed vars against that env.

SAFETY
  • Secrets are entered through a hidden terminal prompt (never via stdin), so an AI
    agent invoking the command never sees the value.
  • No command prints a secret value. Secrets leave the store only by being injected
    into a child process via ` + "`shutup run`" + `.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

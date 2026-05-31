package cmd

import "github.com/spf13/cobra"

// addEnvFlag registers the shared --env flag. An empty value means "use the
// project's default_env" (resolved in the project layer).
func addEnvFlag(cmd *cobra.Command, target *string) {
	cmd.Flags().StringVar(target, "env", "", "environment to target (default: the project's default_env)")
}

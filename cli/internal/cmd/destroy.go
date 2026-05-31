package cmd

import (
	"os"
	"path/filepath"

	"github.com/joe/shutup/internal/agent"
	"github.com/joe/shutup/internal/config"
	"github.com/joe/shutup/internal/tty"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var destroyYes bool

var destroyCmd = &cobra.Command{
	Use:   "destroy",
	Short: "Remove shutup from this project (config + CLAUDE.md block)",
	Long: `Removes shutup from the current project: deletes its shutup.config.yaml and the
shutup block in CLAUDE.md.

It does NOT delete envs — those are shared values that may be referenced by other
projects (and live in ~/.shutup/envs/, outside the repo). Delete an env explicitly
with ` + "`shutup env remove <name> --delete`" + ` only if nothing else uses it. Asks for
confirmation unless --yes.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgPath, err := config.DiscoverCWD()
		if err != nil {
			return err
		}
		claudePath := filepath.Join(filepath.Dir(cfgPath), "CLAUDE.md")

		ui.Info("This will remove shutup from this project:")
		ui.Hint("config:    %s", cfgPath)
		ui.Hint("CLAUDE.md: the shutup block (if present)")
		ui.Hint("envs are NOT deleted (shared; use `shutup env remove --delete` for those)")

		if !destroyYes {
			ans, perr := tty.PromptLine("Type 'yes' to confirm: ")
			if perr != nil {
				return perr
			}
			if ans != "yes" {
				ui.Info("Aborted. Nothing was removed.")
				return nil
			}
		}
		if _, err := agent.RemoveFrom(claudePath); err != nil {
			return err
		}
		if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		ui.Success("Removed shutup from this project (envs kept).")
		return nil
	},
}

func init() {
	destroyCmd.Flags().BoolVar(&destroyYes, "yes", false, "skip the confirmation prompt")
	rootCmd.AddCommand(destroyCmd)
}

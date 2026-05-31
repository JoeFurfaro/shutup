package cmd

import (
	"os"
	"path/filepath"
	"sort"

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

		// Capture the env ids this project references, to report after removal so
		// they can be cleaned up if orphaned.
		var envIDs []string
		if cfg, lerr := config.Load(cfgPath); lerr == nil {
			seen := map[string]bool{}
			for _, envID := range cfg.Envs {
				if !seen[envID] {
					seen[envID] = true
					envIDs = append(envIDs, envID)
				}
			}
			sort.Strings(envIDs)
		}

		ui.Info("This will remove shutup from this project:")
		ui.Hint("config:    %s", cfgPath)
		ui.Hint("CLAUDE.md: the shutup block (if present)")
		ui.Hint("envs are NOT deleted (they may be shared)")

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
		if len(envIDs) > 0 {
			ui.Info("")
			ui.Hint("this project referenced these envs (now possibly orphaned):")
			for _, envID := range envIDs {
				ui.Hint("  %s", envID)
			}
			ui.Hint("delete any that nothing else uses with `shutup env delete <id>`")
		}
		return nil
	},
}

func init() {
	destroyCmd.Flags().BoolVar(&destroyYes, "yes", false, "skip the confirmation prompt")
	rootCmd.AddCommand(destroyCmd)
}

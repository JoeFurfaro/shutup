package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joe/shutup/internal/agent"
	"github.com/joe/shutup/internal/config"
	"github.com/joe/shutup/internal/env"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	initLink       string
	initNoClaudeMD bool
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up shutup in the current directory",
	Long: `Creates shutup.config.yaml in the current directory and gives it a default env
named "dev".

By default a fresh, empty env is created. Use --link <env-id> to point this
project at an EXISTING env instead (sharing it) — find ids with ` + "`shutup env list --all`" + `.

Also writes the shutup instruction block into CLAUDE.md so AI agents know to use
the shutup interface instead of reading .env files.

Refuses if you are already inside a shutup project (a shutup.config.yaml exists in
this directory or any parent).`,
	Args: cobra.NoArgs,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringVar(&initLink, "link", "", "link an existing env id as the default env instead of creating one")
	initCmd.Flags().BoolVar(&initNoClaudeMD, "no-claude-md", false, "do not write the shutup block into CLAUDE.md")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	if existing, derr := config.Discover(cwd); derr == nil {
		return fmt.Errorf(
			"already inside a shutup project (root: %s)\n  run commands from anywhere inside it, or cd elsewhere to start a new one",
			existing)
	} else if !errors.Is(derr, config.ErrNotFound) {
		return derr
	}

	store, err := env.NewLocalEnvStore()
	if err != nil {
		return err
	}

	// Determine the default env: link an existing one, or create fresh.
	var envID string
	if initLink != "" {
		e, lerr := store.Load(initLink)
		if lerr == env.ErrNotFound {
			return fmt.Errorf("no env with id %q on this machine (see `shutup env list --all`)", initLink)
		} else if lerr != nil {
			return lerr
		}
		envID = e.ID
	} else {
		e, cerr := store.Create()
		if cerr != nil {
			return fmt.Errorf("creating env: %w", cerr)
		}
		envID = e.ID
	}

	cfgPath := filepath.Join(cwd, config.Filename)
	cfg := config.New(cfgPath, "dev", envID)
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("writing %s: %w", config.Filename, err)
	}

	res := agent.Unchanged
	wroteClaudeMD := false
	if !initNoClaudeMD {
		var ierr error
		res, ierr = agent.InjectInto(filepath.Join(cwd, "CLAUDE.md"))
		if ierr != nil {
			return fmt.Errorf("updating CLAUDE.md: %w", ierr)
		}
		wroteClaudeMD = res == agent.Created || res == agent.Added || res == agent.Updated
	}

	ui.Success("Initialized shutup project")
	ui.Hint("config:      %s", config.Filename)
	if initLink != "" {
		ui.Hint("default env: dev (linked %s)", ui.Value(envID))
	} else {
		ui.Hint("default env: dev (%s)", ui.Value(envID))
	}
	if initNoClaudeMD {
		ui.Hint("CLAUDE.md:   skipped (--no-claude-md)")
	} else {
		switch res {
		case agent.Created:
			ui.Hint("CLAUDE.md:   created with shutup instructions")
		case agent.Added:
			ui.Hint("CLAUDE.md:   added shutup instructions")
		case agent.Updated:
			ui.Hint("CLAUDE.md:   updated shutup instructions")
		case agent.Unchanged:
			ui.Hint("CLAUDE.md:   already up to date")
		}
	}
	ui.Info("")
	if wroteClaudeMD {
		ui.Info("shutup edited CLAUDE.md (a tracked file) so agents use it instead of reading .env — re-run with --no-claude-md to skip.")
	}
	ui.Hint("next: `shutup set <NAME>` to add a variable (secret by default)")
	return nil
}

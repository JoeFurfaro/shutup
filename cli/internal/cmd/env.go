package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/joe/shutup/internal/env"
	"github.com/joe/shutup/internal/project"
	"github.com/joe/shutup/internal/tty"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environments (the value bags projects consume)",
	Long: `Environments hold the actual values; projects reference them by id under a local
name. These subcommands manage which envs a project uses and let you hand a
secret-free env bundle to a teammate.`,
}

// env add
var (
	envAddLink       string
	envAddCopyFrom   string
	envAddPublicOnly bool
)
var envAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Register a new env in this project (create fresh, --link, or --copy-from another)",
	Long: `Adds an env to the current project under <name>.

By default a fresh, empty env is created. Alternatives:

  --link <env-id>      point this project at an EXISTING env (sharing it) — find
                       ids with ` + "`shutup env list --all`" + `. Two projects then reference
                       the same id; values are stored once.
  --copy-from <name>   seed the new env with a COPY of another env's values from
                       this project (e.g. stand up "prod" from "dev"). Values are
                       copied machine-side and never printed. Add --public-only to
                       copy just the non-secret vars and re-enter secrets per env.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if envAddLink != "" && envAddCopyFrom != "" {
			return fmt.Errorf("--link and --copy-from are mutually exclusive: --link shares an existing env, --copy-from seeds a fresh one")
		}
		if envAddPublicOnly && envAddCopyFrom == "" {
			return fmt.Errorf("--public-only only applies with --copy-from")
		}
		p, err := project.Open()
		if err != nil {
			return err
		}
		if _, exists := p.Config.EnvID(name); exists {
			return fmt.Errorf("this project already has an env named %q", name)
		}
		var envID string
		var publicNames, secretNames []string
		switch {
		case envAddLink != "":
			e, lerr := p.Store.Load(envAddLink)
			if lerr == env.ErrNotFound {
				return fmt.Errorf("no env with id %q on this machine (see `shutup env list --all`)", envAddLink)
			} else if lerr != nil {
				return lerr
			}
			envID = e.ID
		case envAddCopyFrom != "":
			id, pub, sec, cerr := p.CopyEnv(envAddCopyFrom, envAddPublicOnly)
			if cerr != nil {
				return cerr
			}
			envID, publicNames, secretNames = id, pub, sec
		default:
			e, cerr := p.Store.Create()
			if cerr != nil {
				return cerr
			}
			envID = e.ID
		}
		p.Config.Envs[name] = envID
		if p.Config.DefaultEnv == "" {
			p.Config.DefaultEnv = name
		}
		if err := p.Config.Save(); err != nil {
			return err
		}
		if envAddCopyFrom != "" {
			ui.Success("Added env %s (%s) — copied %d vars from %s", ui.Bold(name), ui.Value(envID), len(publicNames)+len(secretNames), ui.Bold(envAddCopyFrom))
			if len(publicNames) > 0 {
				ui.Hint("public: %s", strings.Join(publicNames, ", "))
			}
			if len(secretNames) > 0 {
				ui.Hint("secret: %s", strings.Join(secretNames, ", "))
				ui.Hint("if %s's secrets should differ, overwrite with `shutup set <NAME> --env %s`", name, name)
			}
			return nil
		}
		ui.Success("Added env %s (%s)", ui.Bold(name), ui.Value(envID))
		return nil
	},
}

// env list
var envLsAll bool
var envLsCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List this project's envs (or --all envs on this machine)",
	Long: `Without flags, lists the envs this project references (name -> id). With --all,
lists every env on this machine (id, source, var count) — useful for finding an
id to ` + "`shutup env add <name> --link <id>`" + `.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// --all reads the machine-wide store and is project-agnostic, so it
		// must work outside a project — that's the case where you're hunting
		// for envs orphaned by `shutup destroy`.
		if envLsAll {
			store, err := env.NewLocalEnvStore()
			if err != nil {
				return err
			}
			all, lerr := store.List()
			if lerr != nil {
				return lerr
			}
			if len(all) == 0 {
				ui.Info("No envs on this machine yet.")
				return nil
			}
			for _, e := range all {
				fmt.Printf("%s  %s  %d vars\n", ui.Bold(e.ID), ui.Dim(e.Source), len(e.Vars))
			}
			return nil
		}
		p, err := project.Open()
		if err != nil {
			return err
		}
		if len(p.Config.Envs) == 0 {
			ui.Info("This project references no envs.")
			return nil
		}
		for name, envID := range p.Config.Envs {
			marker := ""
			if name == p.Config.DefaultEnv {
				marker = ui.Dim(" (default)")
			}
			fmt.Printf("%s -> %s%s\n", ui.Bold(name), ui.Value(envID), marker)
		}
		return nil
	},
}

// env default
var envDefaultCmd = &cobra.Command{
	Use:   "default <name>",
	Short: "Set this project's default env",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := project.Open()
		if err != nil {
			return err
		}
		if _, ok := p.Config.EnvID(args[0]); !ok {
			return fmt.Errorf("this project has no env named %q", args[0])
		}
		p.Config.DefaultEnv = args[0]
		if err := p.Config.Save(); err != nil {
			return err
		}
		ui.Success("Default env is now %s", ui.Bold(args[0]))
		return nil
	},
}

// env remove
var envRmDelete bool
var envRmCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm"},
	Short:   "Unmap an env from this project (does not delete shared values by default)",
	Long: `Removes the <name> -> id mapping from this project. The env's values are NOT
deleted by default (other projects may share the env). Pass --delete to also
delete the env bag from this machine (confirmed) — only do this if nothing else
uses it.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		p, err := project.Open()
		if err != nil {
			return err
		}
		envID, ok := p.Config.EnvID(name)
		if !ok {
			return fmt.Errorf("this project has no env named %q", name)
		}
		if name == p.Config.DefaultEnv && len(p.Config.Envs) > 1 {
			return fmt.Errorf("%q is the default env; set another default first (`shutup env default <name>`)", name)
		}
		delete(p.Config.Envs, name)
		if name == p.Config.DefaultEnv {
			p.Config.DefaultEnv = ""
		}
		if err := p.Config.Save(); err != nil {
			return err
		}
		ui.Success("Unmapped env %s from this project", ui.Bold(name))
		if envRmDelete {
			ans, perr := tty.PromptLine(fmt.Sprintf("Delete the env bag %s from this machine? other projects may use it. Type 'yes': ", envID))
			if perr != nil {
				return perr
			}
			if ans == "yes" {
				if err := p.Store.Delete(envID); err != nil {
					return err
				}
				ui.Success("Deleted env bag %s", ui.Value(envID))
			} else {
				ui.Info("Kept the env bag.")
			}
		}
		return nil
	},
}

// env delete (global, by id — no project required; for cleaning up orphans)
var envDeleteYes bool
var envDeleteCmd = &cobra.Command{
	Use:   "delete <env-id>",
	Short: "Delete an env bag from this machine by id (e.g. orphans left after destroy)",
	Long: `Deletes an env's stored values from this machine by id. Unlike ` + "`env remove`" + ` (which
unmaps an env from the current project), this operates directly on the store and
needs no project — so it's how you clean up envs orphaned after ` + "`shutup destroy`" + `.

Find ids with ` + "`shutup env list --all`" + `. This is destructive and cannot be undone;
asks for confirmation unless --yes. Make sure no other project still uses the id.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		envID := args[0]
		store, err := env.NewLocalEnvStore()
		if err != nil {
			return err
		}
		if _, lerr := store.Load(envID); lerr == env.ErrNotFound {
			return fmt.Errorf("no env with id %q on this machine", envID)
		} else if lerr != nil {
			return lerr
		}
		if !envDeleteYes {
			ans, perr := tty.PromptLine(fmt.Sprintf("Permanently delete env %s and its values? Type 'yes': ", envID))
			if perr != nil {
				return perr
			}
			if ans != "yes" {
				ui.Info("Aborted. Nothing was deleted.")
				return nil
			}
		}
		if err := store.Delete(envID); err != nil {
			return err
		}
		ui.Success("Deleted env %s", ui.Value(envID))
		return nil
	},
}

// env export
var envExportOut string
var envExportCmd = &cobra.Command{
	Use:   "export [name]",
	Short: "Write a SECRET-FREE env bundle to share with a teammate",
	Long: `Writes a bundle for the env (default: the project's default_env) that is safe to
hand to a teammate out-of-band (file, Slack, email). The bundle contains the env
id, its public values, and the NAMES of its secrets — but NEVER secret values.

The recipient runs ` + "`shutup env import <file>`" + `, then fills the secrets locally. Because
the bundle carries the same env id, you both end up on the same logical env.

Writes to --out <file>, or stdout if omitted.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p, err := project.Open()
		if err != nil {
			return err
		}
		name := ""
		if len(args) == 1 {
			name = args[0]
		}
		e, err := p.ResolveEnv(name)
		if err != nil {
			return err
		}
		data, err := env.MarshalBundle(e)
		if err != nil {
			return err
		}
		if envExportOut == "" {
			fmt.Print(string(data))
			return nil
		}
		if err := os.WriteFile(envExportOut, data, 0o644); err != nil {
			return err
		}
		ui.Success("Wrote secret-free bundle to %s", ui.Value(envExportOut))
		ui.Hint("safe to share — contains public values + secret names, no secret values")
		return nil
	},
}

// env import
var envImportAs string
var envImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import a teammate's env bundle (merges by id, keeps your local secrets)",
	Long: `Imports a secret-free env bundle (from ` + "`shutup env export`" + `) into this machine's
env store, under the bundle's id. Public values are applied; secret names become
unset placeholders; any secret values you already had locally are preserved.

With --as <name>, also maps the imported env into the current project under that
name (otherwise it just lands in your store and you link it with
` + "`shutup env add <name> --link <id>`" + `). Then run ` + "`shutup missing`" + ` and set the secrets.

(Distinct from the top-level ` + "`shutup import <.env>`" + `, which migrates a raw .env file.)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		bundle, err := env.UnmarshalBundle(data)
		if err != nil {
			return err
		}
		store, err := env.NewLocalEnvStore()
		if err != nil {
			return err
		}
		merged, err := store.ImportBundle(bundle)
		if err != nil {
			return err
		}
		ui.Success("Imported env %s", ui.Value(merged.ID))
		if envImportAs != "" {
			p, perr := project.Open()
			if perr != nil {
				return fmt.Errorf("--as given but not inside a project: %w", perr)
			}
			if _, exists := p.Config.EnvID(envImportAs); exists {
				return fmt.Errorf("this project already has an env named %q", envImportAs)
			}
			p.Config.Envs[envImportAs] = merged.ID
			if p.Config.DefaultEnv == "" {
				p.Config.DefaultEnv = envImportAs
			}
			if err := p.Config.Save(); err != nil {
				return err
			}
			ui.Hint("linked as %s in this project — run `shutup missing` to see secrets to set", envImportAs)
		} else {
			ui.Hint("link it into a project: `shutup env add <name> --link %s`", merged.ID)
		}
		return nil
	},
}

func init() {
	envAddCmd.Flags().StringVar(&envAddLink, "link", "", "link an existing env id instead of creating a new env")
	envAddCmd.Flags().StringVar(&envAddCopyFrom, "copy-from", "", "seed the new env with a copy of another project env's values (by name)")
	envAddCmd.Flags().BoolVar(&envAddPublicOnly, "public-only", false, "with --copy-from, copy only public vars (re-enter secrets per env)")
	envLsCmd.Flags().BoolVar(&envLsAll, "all", false, "list every env on this machine, not just this project's")
	envRmCmd.Flags().BoolVar(&envRmDelete, "delete", false, "also delete the env bag from this machine (confirmed)")
	envExportCmd.Flags().StringVarP(&envExportOut, "out", "o", "", "write the bundle to a file instead of stdout")
	envImportCmd.Flags().StringVar(&envImportAs, "as", "", "also map the imported env into this project under this name")

	envDeleteCmd.Flags().BoolVar(&envDeleteYes, "yes", false, "skip the confirmation prompt")

	envCmd.AddCommand(envAddCmd, envLsCmd, envDefaultCmd, envRmCmd, envDeleteCmd, envExportCmd, envImportCmd)
	rootCmd.AddCommand(envCmd)
}

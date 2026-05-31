package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/joe/shutup/internal/dotenv"
	"github.com/joe/shutup/internal/project"
	"github.com/joe/shutup/internal/tty"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	importEnv         string
	importPublic      []string
	importInteractive bool
	importDelete      bool
)

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Migrate a .env file into an env (classify by name, never blind)",
	Long: `Migrates an existing .env file into the target env (default: the project's
default_env), wiring the imported names into this project's consumes.

DISCOVERY FIRST — a bare ` + "`shutup import <file>`" + ` does NOT import; it only prints the
variable NAMES found, so you (or an agent) can classify them WITHOUT reading the
file's values into context. To actually import you must make a choice:

  shutup import .env --public PORT,LOG_LEVEL   # those public, everything else secret
  shutup import .env -i                        # interactively classify each, one by one

Values flow file -> env directly (never printed). Add --delete to remove the .env
afterward. (For exchanging a shutup env between people, use ` + "`shutup env export/import`" + `.)`,
	Args: cobra.ExactArgs(1),
	RunE: runImport,
}

func init() {
	addEnvFlag(importCmd, &importEnv)
	importCmd.Flags().StringSliceVar(&importPublic, "public", nil, "comma-separated names to mark public (everything else is secret)")
	importCmd.Flags().BoolVarP(&importInteractive, "interactive", "i", false, "classify each variable one by one (human)")
	importCmd.Flags().BoolVar(&importDelete, "delete", false, "delete the source file after a successful import")
	rootCmd.AddCommand(importCmd)
}

func runImport(cmd *cobra.Command, args []string) error {
	file := args[0]
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	pairs, err := dotenv.Parse(f)
	f.Close()
	if err != nil {
		return err
	}
	if len(pairs) == 0 {
		ui.Info("No variables found in %s.", file)
		return nil
	}

	classifyByFlag := cmd.Flags().Changed("public")

	// Discovery mode: no classification choice made -> just list names, never import.
	if !classifyByFlag && !importInteractive {
		for _, p := range pairs {
			fmt.Println(p.Name)
		}
		ui.Info("")
		ui.Hint("found %d variables. Classify and import with one of:", len(pairs))
		ui.Hint("`shutup import %s --public <names>`   (rest are secret)", file)
		ui.Hint("`shutup import %s -i`                 (interactive)", file)
		return nil
	}

	p, err := project.Open()
	if err != nil {
		return err
	}

	publicSet := map[string]bool{}
	for _, n := range importPublic {
		publicSet[strings.TrimSpace(n)] = true
	}

	imported := 0
	var publicNames []string
	for _, pair := range pairs {
		public := publicSet[pair.Name]
		if importInteractive {
			choice, perr := tty.PromptLine(fmt.Sprintf("%s = %s   [s]ecret / [p]ublic? (s) ", ui.Bold(pair.Name), ui.Dim(preview(pair.Value))))
			if perr != nil {
				return perr
			}
			public = strings.EqualFold(strings.TrimSpace(choice), "p")
		}
		if _, err := p.SetVar(importEnv, pair.Name, pair.Value, public); err != nil {
			return err
		}
		if public {
			publicNames = append(publicNames, pair.Name)
		}
		imported++
	}

	ui.Success("Imported %d variables into env %s", imported, ui.Value(effectiveEnv(p, importEnv)))
	// Echo what was made public (names only) so a mis-classified secret is
	// visible for review — the rest are secret by default.
	if len(publicNames) > 0 {
		ui.Hint("marked public: %s", strings.Join(publicNames, ", "))
	}
	if importDelete {
		if err := os.Remove(file); err != nil {
			return err
		}
		ui.Hint("deleted %s", file)
	} else {
		ui.Hint("the plaintext %s is still on disk — re-run with --delete to remove it", file)
	}
	return nil
}

func preview(v string) string {
	if len(v) > 4 {
		return v[:4] + "…"
	}
	return v
}

func effectiveEnv(p *project.Project, name string) string {
	if name != "" {
		return name
	}
	return p.Config.DefaultEnv
}

package cmd

import (
	"fmt"

	"github.com/joe/shutup/internal/project"
	"github.com/joe/shutup/internal/tty"
	"github.com/joe/shutup/internal/ui"
	"github.com/spf13/cobra"
)

var (
	setEnv    string
	setPublic bool
)

var setCmd = &cobra.Command{
	Use:   "set <NAME> [value]",
	Short: "Set a variable's value in an env (and consume it in this project)",
	Long: `Writes a variable's value into the target env, and ensures the current project
consumes it (auto-wired if new).

Secrets (the default) are entered through a HIDDEN terminal prompt, so the value
never appears on screen or in an AI agent's context:

    shutup set STRIPE_KEY

Public (non-sensitive) values may be passed inline, but only with --public:

    shutup set PORT 3000 --public

An inline value WITHOUT --public is refused: an inline value is visible to anyone
(or any agent) reading the command, so it must not carry a secret. The value is
written to --env (default: the project's default_env). Visibility (secret/public)
is stored on the var in the env. This command never prints the value back.

AI AGENTS: you cannot set a SECRET yourself — the hidden prompt needs an
interactive terminal you don't have, and this will refuse. Ask the user to run
"shutup set <NAME>" in their own terminal. You CAN set public values inline
(--public), and you can use secrets afterward via "shutup run".`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runSet,
}

func init() {
	addEnvFlag(setCmd, &setEnv)
	setCmd.Flags().BoolVar(&setPublic, "public", false, "mark the variable as public (non-sensitive)")
	rootCmd.AddCommand(setCmd)
}

func runSet(cmd *cobra.Command, args []string) error {
	name := args[0]
	hasInline := len(args) == 2

	p, err := project.Open()
	if err != nil {
		return err
	}

	var value string
	if hasInline {
		if !setPublic {
			return fmt.Errorf(
				"refusing to set %q from an inline value: it would be visible to anyone "+
					"reading the command (including an AI agent).\n"+
					"  secrets must be entered via the hidden prompt — run `shutup set %s` with no value.\n"+
					"  if this really isn't sensitive, pass --public.",
				name, name)
		}
		value = args[1]
	} else {
		if setPublic {
			return fmt.Errorf("public variable %q needs a value: `shutup set %s <value> --public`", name, name)
		}
		v, perr := tty.PromptHidden(fmt.Sprintf("Enter value for %s (hidden from your AI agent): ", name))
		if perr == tty.ErrNoTerminal {
			return fmt.Errorf(
				"setting the secret %q needs an interactive terminal (it never reads stdin).\n"+
					"  if you're an AI agent: do NOT try to set this yourself — ask the user to run\n"+
					"    shutup set %s\n"+
					"  in their own terminal. You can still use the secret afterward via `shutup run`.",
				name, name)
		}
		if perr != nil {
			return perr
		}
		if v == "" {
			return fmt.Errorf("no value entered; nothing stored")
		}
		value = v
	}

	wired, err := p.SetVar(setEnv, name, value, setPublic)
	if err != nil {
		return err
	}

	kind := "secret"
	if setPublic {
		kind = "public"
	}
	envLabel := setEnv
	if envLabel == "" {
		envLabel = p.Config.DefaultEnv
	}
	if setPublic {
		ui.Success("Set %s=%s (%s) in env %s", ui.Bold(name), ui.Value(value), kind, ui.Value(envLabel))
	} else {
		ui.Success("Stored %s (%s) in env %s", ui.Bold(name), kind, ui.Value(envLabel))
	}
	if wired {
		ui.Hint("wired %s into this project's consumes", name)
	}
	return nil
}

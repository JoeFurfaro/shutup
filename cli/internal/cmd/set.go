package cmd

import (
	"fmt"

	"github.com/joe/shutup/internal/project"
	"github.com/joe/shutup/internal/securedialog"
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

AI AGENTS: you CAN run this for a secret. With no terminal, it opens a native
secure dialog on the user's screen for them to type into — the value goes
straight to the store and never reaches you (you only see "Stored"). If there's
no display either (headless), it refuses and asks you to have the user run it in
a terminal. Either way you never see the value.`,
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
		v, perr := promptSecretValue(name)
		if perr != nil {
			return perr
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

// promptSecretValue collects a secret value without ever exposing it to a calling
// agent, trying the most natural channel available:
//  1. the controlling terminal (hidden /dev/tty prompt) — when run by a human,
//  2. a native secure dialog (OS window server) — when there's no terminal but a
//     desktop display (e.g. an agent ran the command on your machine),
//  3. otherwise refuse and tell the caller to run it in a terminal (headless).
//
// In every case the value goes straight into shutup's memory → store; it is never
// printed, never on the command line, never on stdin.
func promptSecretValue(name string) (string, error) {
	label := fmt.Sprintf("Enter value for %s (hidden from your AI agent): ", name)

	v, err := tty.PromptHidden(label)
	switch {
	case err == nil:
		// got it from the terminal
	case err == tty.ErrNoTerminal:
		// No terminal — try a native secure dialog (agent-on-desktop case).
		ui.Info("Opening a secure input dialog — enter the value there…")
		dv, derr := securedialog.Prompt(label)
		switch {
		case derr == nil:
			v = dv
		case securedialog.IsCanceled(derr):
			return "", fmt.Errorf("canceled; nothing stored")
		default:
			// No dialog either (headless) — guide the caller.
			return "", fmt.Errorf(
				"setting the secret %q needs an interactive terminal or a desktop session.\n"+
					"  if you're an AI agent in a headless context: ask the user to run\n"+
					"    shutup set %s\n"+
					"  in their own terminal. Use the secret afterward via `shutup run`.",
				name, name)
		}
	default:
		return "", err
	}

	if v == "" {
		return "", fmt.Errorf("no value entered; nothing stored")
	}
	return v, nil
}

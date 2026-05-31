package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/joe/shutup/internal/project"
	"github.com/spf13/cobra"
)

var runEnv string

var runCmd = &cobra.Command{
	Use:   "run -- <command> [args...]",
	Short: "Run a command with this project's consumed variables injected",
	Long: `Injects ONLY the variables this project consumes — resolved against the target
env (default: the project's default_env) — into a child process, then runs it.

The child receives the values (secrets included) in its environment; you never
see them. Only consumed vars are injected (least-privilege), so a project never
leaks variables it doesn't use. Stdin/stdout/stderr and Ctrl+C are forwarded, and
the child's exit code is propagated.

    shutup run -- npm start
    shutup run --env prod -- ./server`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRun,
}

func init() {
	addEnvFlag(runCmd, &runEnv)
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	p, err := project.Open()
	if err != nil {
		return err
	}
	childEnv, err := p.ChildEnv(runEnv)
	if err != nil {
		return err
	}

	c := exec.Command(args[0], args[1:]...)
	c.Env = childEnv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return fmt.Errorf("starting %q: %w", args[0], err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		for s := range sigCh {
			_ = c.Process.Signal(s)
		}
	}()

	err = c.Wait()
	if err == nil {
		return nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			os.Exit(128 + int(ws.Signal()))
		}
		code := exitErr.ExitCode()
		if code < 0 {
			code = 1
		}
		os.Exit(code)
	}
	return err
}

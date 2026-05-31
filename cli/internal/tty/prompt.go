// Package tty implements the TTY-bypass: reading a secret directly from the
// controlling terminal with echo disabled, NOT from stdin.
//
// This is the core safety mechanism. When an AI agent invokes `shutup set`, the
// agent owns the process's stdin/stdout — but the value the user types flows
// keyboard -> /dev/tty -> this process, never through stdin and never into the
// agent's captured output. We deliberately do NOT fall back to stdin if no
// terminal is available; that would defeat the entire guarantee.
//
// The package exposes one primitive, PromptHidden. v1 implements the Unix path
// (/dev/tty); the Windows equivalent (the CONIN$ console handle) slots in behind
// the same function later without callers changing.
package tty

import "errors"

// ErrNoTerminal is returned when there is no controlling terminal to read from.
var ErrNoTerminal = errors.New(
	"no terminal available for hidden input — shutup never reads secrets from stdin " +
		"(that would expose them to an AI agent's context); run this where you can type")

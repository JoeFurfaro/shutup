// Package ui centralizes human-facing output styling.
//
// Color is emitted only when the target stream is a real terminal (fatih/color
// auto-detects this and honors NO_COLOR). That matters here beyond aesthetics:
// shutup is designed to be invoked by AI agents that capture our output, and we
// must never spray ANSI escape codes into an agent's context. Status lines and
// prompts go to stderr; machine/human-readable data goes to stdout.
package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	green = color.New(color.FgGreen)
	red   = color.New(color.FgRed)
	dim   = color.New(color.Faint)
	bold  = color.New(color.Bold)
	cyan  = color.New(color.FgCyan)
)

// Success prints a green check line to stderr.
func Success(format string, a ...any) {
	green.Fprint(os.Stderr, "✓ ")
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Errorf prints a red error line to stderr.
func Errorf(format string, a ...any) {
	red.Fprint(os.Stderr, "✗ ")
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Hint prints a dim, indented suggestion line to stderr.
func Hint(format string, a ...any) {
	dim.Fprintf(os.Stderr, "  "+format+"\n", a...)
}

// Info prints a plain line to stderr.
func Info(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
}

// Bold returns name styled bold (for variable names).
func Bold(s string) string { return bold.Sprint(s) }

// Value returns s styled as a value (cyan).
func Value(s string) string { return cyan.Sprint(s) }

// Dim returns s styled faint (for redaction markers, secondary text).
func Dim(s string) string { return dim.Sprint(s) }

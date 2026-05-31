// Package securedialog shows a native, masked secure-input dialog so a secret
// can be entered when there is no interactive terminal (e.g. an AI agent ran the
// command) but a desktop display is available.
//
// The typed value is returned to the caller in-process and is never printed —
// the same isolation property as the /dev/tty prompt, achieved via the OS window
// server instead of the terminal. The agent that spawned the command only ever
// sees the command's stdout/stderr, which never carries the value.
//
// Cross-platform and cgo-free via ncruces/zenity: osascript on macOS, zenity/
// kdialog on Linux, native dialogs on Windows. If no display is available the
// underlying call returns an error and the caller falls back to a terminal prompt.
package securedialog

import (
	"errors"

	"github.com/ncruces/zenity"
)

// Prompt shows a masked entry dialog (titled "shutup") with the given label and
// returns the entered value. Returns an error wrapping zenity.ErrCanceled if the
// user dismissed it, or another error if no dialog could be shown (no display).
func Prompt(label string) (string, error) {
	return zenity.Entry(label,
		zenity.Title("shutup"),
		zenity.HideText(),
	)
}

// IsCanceled reports whether err is the user-dismissed-the-dialog sentinel.
func IsCanceled(err error) bool {
	return errors.Is(err, zenity.ErrCanceled)
}

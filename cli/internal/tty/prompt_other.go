//go:build !unix

package tty

// PromptHidden is not implemented on non-Unix platforms in v1. The Windows path
// (open the CONIN$ console handle, disable echo via x/term) slots in here later.
func PromptHidden(label string) (string, error) {
	return "", ErrNoTerminal
}

// PromptLine is not implemented on non-Unix platforms in v1.
func PromptLine(label string) (string, error) {
	return "", ErrNoTerminal
}

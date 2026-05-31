//go:build unix

package tty

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// PromptHidden writes label to the controlling terminal, reads a line with echo
// disabled, and returns it. It reads from /dev/tty directly — never stdin.
func PromptHidden(label string) (string, error) {
	t, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", ErrNoTerminal
	}
	defer t.Close()

	// Prompt is written to the tty itself (not stderr) so the human always sees
	// it on screen even when stderr is captured by an agent.
	fmt.Fprint(t, label)
	b, err := term.ReadPassword(int(t.Fd()))
	fmt.Fprintln(t) // newline after the (hidden) input
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// PromptLine writes label to the controlling terminal and reads one line with
// echo ON (for visible confirmations, not secrets). Reads /dev/tty, never stdin.
func PromptLine(label string) (string, error) {
	t, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return "", ErrNoTerminal
	}
	defer t.Close()

	fmt.Fprint(t, label)
	line, err := bufio.NewReader(t).ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// Package agent manages the instruction block shutup writes into agent context
// files (CLAUDE.md today; Cursor/AGENTS.md/Codex later). The block is what makes
// an AI assistant choose the safe path — using the shutup interface instead of
// reading .env files or echoing secret values into its context.
package agent

import (
	"os"
	"strings"
)

const (
	startMarker = "<!-- shutup:start -->"
	endMarker   = "<!-- shutup:end -->"
)

// Block is the instruction text injected between the markers. It presents the
// complete interface as "yours to use" (every command is agent-safe), explains
// the why briefly, and states the one hard rule positively.
const Block = `## Environment variables & secrets — managed by ` + "`shutup`" + `

This project uses **shutup** for all environment variables and secrets. The whole point
is that you (the AI assistant) can *use* secrets without ever *seeing* their values —
keeping them out of this conversation, out of logs, and out of anywhere they could leak.

**Mental model:** an *environment* (e.g. ` + "`dev`" + `) is a bag of variables (each value +
secret/public flag) stored on the machine (` + "`~/.shutup/envs/`" + `), not in the repo. This
*project* just declares the variable NAMES it consumes. You work from inside the project;
` + "`--env`" + ` picks the env (default: the project's default).

**Core rule:** never handle raw secret values. Don't ask the user to paste a secret into
the chat, don't read ` + "`.env`" + ` files, and don't ` + "`cat`, `echo`, or `printenv`" + ` a secret. Always
go through the commands below — this is how configuration works here, and it's what keeps
the user safe. Every command below is safe for you to run.

### How to do what you need

- **Set a SECRET — run ` + "`shutup set <NAME>`" + `.** With no terminal, it opens a native secure
  dialog on the user's screen; they type the value there and it goes straight to the
  store — you only ever see "Stored", never the value (that's intended). If there's no
  display (headless), it will tell you to have the user run it in their terminal. Never
  ask the user to paste a secret into the chat.
- **Set a public (non-sensitive) value** — ` + "`shutup set <NAME> <value> --public`" + `.
  Both forms wire the var into this project's consumes.

- **Consume an already-set variable** — ` + "`shutup use <NAME>`" + `
  Declares this project needs ` + "`<NAME>`" + ` (e.g. a value already set in a shared env), no value.

- **See what's not set yet** — ` + "`shutup missing`" + `
  Lists consumed vars with no value in the env (one name per line). Drive setup with this.

- **Check existence (no value)** — ` + "`shutup check <NAME>`" + ` → ` + "`exists`/`not found`" + ` (exit 0/1).

- **See state** — ` + "`shutup list`" + ` → public values shown; secrets only as ` + "`(secret, set)`" + `.

- **Run with values injected** — ` + "`shutup run -- <command>`" + ` (e.g. ` + "`shutup run -- npm start`" + `)
  Injects only this project's consumed vars; the program gets the secrets, you never do.

- **Migrate an existing ` + "`.env`" + `** — run ` + "`shutup import <file>`" + ` FIRST with no flags: it
  only lists the variable names (it does NOT read values into your context). Classify by
  name, then ` + "`shutup import <file> --public <names>`" + ` (rest are secret), optionally ` + "`--delete`" + `.
  Never ` + "`cat`" + ` the ` + "`.env`" + ` yourself.

### When the user asks you to set up secrets

1. ` + "`shutup missing`" + ` (or, if migrating, ` + "`shutup import <file>`" + ` to list names).
2. Run ` + "`shutup set <NAME>`" + ` for each missing SECRET — a secure dialog opens for the user to
   fill (you never see the value). Set non-secrets yourself with ` + "`--public`" + `.
3. Run/test with ` + "`shutup run -- <command>`" + ` — the process gets the values, you never do.

If you think you need a secret's actual value, you almost certainly don't: use
` + "`shutup check`" + ` to confirm it exists and ` + "`shutup run`" + ` to use it.`

// section is the full marked block, including markers and surrounding blank lines.
func section() string {
	return startMarker + "\n" + Block + "\n" + endMarker
}

// WriteResult reports what InjectInto did.
type WriteResult int

const (
	Created WriteResult = iota // file did not exist; created it with the block
	Added                      // file existed; appended the block
	Updated                    // file had an old block; replaced it in place
	Unchanged                  // block already present and identical
)

// InjectInto writes (or updates) the shutup block in the file at path,
// idempotently. An existing block between the markers is replaced in place;
// otherwise the block is appended; a missing file is created.
func InjectInto(path string) (WriteResult, error) {
	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Created, os.WriteFile(path, []byte(section()+"\n"), 0o644)
	}
	if err != nil {
		return 0, err
	}

	content := string(existing)
	start := strings.Index(content, startMarker)
	end := strings.Index(content, endMarker)

	if start >= 0 && end > start {
		// Replace the existing block in place.
		newContent := content[:start] + section() + content[end+len(endMarker):]
		if newContent == content {
			return Unchanged, nil
		}
		return Updated, os.WriteFile(path, []byte(newContent), 0o644)
	}

	// Append, ensuring a blank line of separation.
	sep := "\n"
	if !strings.HasSuffix(content, "\n") {
		sep = "\n\n"
	} else if !strings.HasSuffix(content, "\n\n") {
		sep = "\n"
	}
	return Added, os.WriteFile(path, []byte(content+sep+section()+"\n"), 0o644)
}

// RemoveFrom strips the shutup block (and its markers) from the file at path.
// Returns whether a block was found and removed. A missing file is not an error.
func RemoveFrom(path string) (bool, error) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	s := string(content)
	start := strings.Index(s, startMarker)
	end := strings.Index(s, endMarker)
	if start < 0 || end < start {
		return false, nil
	}
	cleaned := strings.TrimRight(s[:start], "\n") + s[end+len(endMarker):]
	cleaned = strings.TrimLeft(cleaned, "\n")
	if cleaned != "" && !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}
	return true, os.WriteFile(path, []byte(cleaned), 0o644)
}

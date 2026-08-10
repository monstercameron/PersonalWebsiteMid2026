//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// A deep link is what makes the terminal reachable from outside itself: a link in an application
// email can land a recruiter on `cat notes/experience.md` instead of on an empty prompt they have
// to be taught to use.
//
// The command in the URL is attacker-controlled — anyone can send anyone a link — so it is checked
// against an allowlist of read-only commands rather than handed to the shell. The blast radius of
// the alternative is small (the filesystem is a per-visitor localStorage sandbox that repairSeed
// restores) but "small" is not a reason to auto-run `rm -rf ~` because a stranger sent a URL.

// deepLinkParam is the query parameter carrying an autorun command.
const deepLinkParam = "cmd"

// deepLinkSafe is the set of commands a URL may run without the visitor typing anything. Every one
// of them only reads: nothing here writes the filesystem, sends a message, or spends server budget.
var deepLinkSafe = map[string]bool{
	"help": true, "about": true, "whoami": true, "projects": true, "open": true,
	"neofetch": true, "links": true, "resume": true, "anime": true, "tour": true,
	"stats": true, "uptime": true, "ls": true, "cat": true, "pwd": true, "man": true,
	"theme": true, "bench": true,
}

// deepLinkCommand returns the command requested by the current URL, or "".
func deepLinkCommand() string {
	search := js.Global().Get("location").Get("search")
	if !search.Truthy() {
		return ""
	}
	params := js.Global().Get("URLSearchParams").New(search.String())
	v := params.Call("get", deepLinkParam)
	if !v.Truthy() {
		return ""
	}
	return strings.TrimSpace(v.String())
}

// deepLinkAllowed reports whether cmd is safe to run unprompted. Shell operators are rejected
// outright: a pipeline is only as safe as its least safe stage, and allowing `>` would let a link
// overwrite files in the visitor's sandbox.
func deepLinkAllowed(cmd string) bool {
	if cmd == "" || strings.ContainsAny(cmd, "|>&;`$") {
		return false
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	return deepLinkSafe[fields[0]]
}

// runDeepLink runs the URL's command, if it has a safe one, and expands the terminal — a visitor
// who followed a link into the terminal came for the terminal, not for the page around it.
// A rejected command is reported rather than silently ignored, so a mistyped link looks like a
// mistake instead of a broken site.
func runDeepLink(t *termCtl, expanded ui.State[bool]) {
	cmd := deepLinkCommand()
	if cmd == "" {
		return
	}
	if !deepLinkAllowed(cmd) {
		t.emit(gap())
		t.emitRows([]progRow{
			{text: "link asked to run: " + cmd, style: rowErr},
			dim("only read-only commands run automatically from a link. Type it yourself if you meant it."),
		})
		return
	}
	expanded.Set(true)
	// After the boot lines, and after the expand has painted, so the visitor sees the command
	// arrive in a terminal that is already open rather than a burst of text in a small box.
	after(450, func() {
		t.emit(gap())
		runCommand(cmd, t)
	})
}

// runShare implements the `share` command: it copies a URL that reruns a command. With no argument
// it shares the last command the visitor ran, which is the common case — "show someone what I just
// found" — rather than making them retype it.
func runShare(args []string, t *termCtl) {
	cmd := strings.Join(args, " ")
	if cmd == "" {
		cmd = lastShareableCommand(t)
	}
	if cmd == "" {
		t.emitRows([]progRow{
			{text: "share: nothing to share yet", style: rowErr},
			dim("run something first, or name it: share \"open cashflux\""),
		})
		t.emit(gap())
		return
	}
	if !deepLinkAllowed(cmd) {
		t.emitRows([]progRow{
			{text: "share: `" + cmd + "` will not autorun from a link", style: rowErr},
			dim("only read-only commands do — the link would open the terminal and stop there."),
		})
		t.emit(gap())
		return
	}
	url := shareURL(cmd)
	copyToClipboard(url)
	t.emitRows([]progRow{
		row("copied to your clipboard:"),
		link(url, url),
		dim("opening that link boots the terminal and runs `" + cmd + "`."),
	})
	t.emit(gap())
}

// shareURL builds the deep link for a command against the current origin and path.
func shareURL(cmd string) string {
	loc := js.Global().Get("location")
	encode := js.Global().Get("encodeURIComponent")
	return loc.Get("origin").String() + loc.Get("pathname").String() +
		"?" + deepLinkParam + "=" + encode.Invoke(cmd).String()
}

// lastShareableCommand returns the most recent history entry that a link could rerun, skipping
// `share` itself so `share` twice in a row does not share the act of sharing.
func lastShareableCommand(t *termCtl) string {
	if t.sh == nil {
		return ""
	}
	entries := t.sh.nav.entries
	for i := len(entries) - 1; i >= 0; i-- {
		if strings.HasPrefix(entries[i], "share") {
			continue
		}
		if deepLinkAllowed(entries[i]) {
			return entries[i]
		}
	}
	return ""
}

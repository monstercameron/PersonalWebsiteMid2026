//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// maxScrollback caps how many rendered lines the terminal keeps. A visitor who runs `tour`, then
// `matrix`, then pipes a few files around can otherwise grow the DOM without bound in a session
// that never reloads — and a real terminal has a scrollback limit for the same reason.
const maxScrollback = 600

// termCtl is the handle a command writes through. Commands used to return their whole output as a
// value, which meant everything had to be ready before anything appeared: no progress, no typing
// animation, and no way to await the network. Writing through a control instead lets output arrive
// over time, which is what `tour`, `bench`, `stats` and `contact` all need.
type termCtl struct {
	scrollback ui.State[[]ui.Node]
	input      ui.State[string]
	sh         *shell

	// capture, when set, receives the next submitted line instead of the command runner. This is
	// how the interactive `contact` form reads a field at a time without a second input element.
	capture ui.Ref[func(*termCtl, string)]
	// promptOverride replaces the shell sigil while a capture is active (e.g. "your name").
	promptOverride ui.State[string]

	// bootLive is true only while the scrollback is still exactly the boot block. The measured boot
	// numbers arrive asynchronously, and without this a `clear` (or a fast first command) followed
	// by a slow ping would rewrite line 0 of whatever the visitor is now looking at.
	bootLive ui.Ref[bool]

	// gen is bumped on every new command. Long-running output (the tour's typing, an animation)
	// captures the value at its start and stops as soon as it changes, so starting a new command
	// cancels the previous one instead of interleaving two streams into the scrollback.
	gen ui.Ref[int]
}

// emit appends nodes to the scrollback, trimming the oldest lines past maxScrollback.
func (t *termCtl) emit(nodes ...ui.Node) {
	if len(nodes) == 0 {
		return
	}
	t.scrollback.Update(func(prev []ui.Node) []ui.Node {
		next := make([]ui.Node, 0, len(prev)+len(nodes))
		next = append(next, prev...)
		next = append(next, nodes...)
		if len(next) > maxScrollback {
			next = next[len(next)-maxScrollback:]
		}
		return next
	})
}

// emitRows appends portfolio-program rows as styled output.
func (t *termCtl) emitRows(rows []progRow) { t.emit(renderRows(rows)...) }

// replaceLast swaps the final scrollback node, which is how a progress line updates in place
// (a spinner, a benchmark counting up) instead of printing a new line per tick.
func (t *termCtl) replaceLast(node ui.Node) {
	t.scrollback.Update(func(prev []ui.Node) []ui.Node {
		if len(prev) == 0 {
			return []ui.Node{node}
		}
		next := append([]ui.Node{}, prev...)
		next[len(next)-1] = node
		return next
	})
}

// replaceAt swaps the scrollback node at index i, used by the boot log to fill in a measured
// value on the line it belongs to. It is a no-op once the boot block is no longer what is on
// screen, or if the index is out of range.
func (t *termCtl) replaceAt(i int, node ui.Node) {
	if !t.bootLive.Get() {
		return
	}
	t.scrollback.Update(func(prev []ui.Node) []ui.Node {
		if i < 0 || i >= len(prev) {
			return prev
		}
		next := append([]ui.Node{}, prev...)
		next[i] = node
		return next
	})
}

// token returns the current command generation; pass it to live to test for cancellation.
func (t *termCtl) token() int { return t.gen.Get() }

// live reports whether the command that captured tok is still the current one.
func (t *termCtl) live(tok int) bool { return t.gen.Get() == tok }

// beginCapture switches the input line into interactive mode: label is shown in place of the
// prompt, and the next submitted line goes to fn.
func (t *termCtl) beginCapture(label string, fn func(*termCtl, string)) {
	t.promptOverride.Set(label)
	t.capture.Set(fn)
}

// endCapture returns the input line to the normal shell prompt.
func (t *termCtl) endCapture() {
	t.promptOverride.Set("")
	t.capture.Set(nil)
}

// after runs fn once, ms milliseconds from now, releasing the JS callback afterwards. Every
// animated command in the terminal is built from this rather than a blocking sleep: blocking in
// wasm freezes the browser's event loop, taking the whole page down with it.
func after(ms int, fn func()) {
	var cb js.Func
	cb = js.FuncOf(func(js.Value, []js.Value) any {
		defer cb.Release()
		fn()
		return nil
	})
	js.Global().Call("setTimeout", cb, ms)
}

// runCommand is the terminal's entry point for a submitted line. It records history, echoes the
// line, and routes to the interactive capture, a special/async program, a styled portfolio
// program, or the shell — in that order.
func runCommand(raw string, t *termCtl) {
	// Any submission supersedes whatever was still printing.
	t.gen.Set(t.gen.Get() + 1)

	if fn := t.capture.Get(); fn != nil {
		t.emit(capturedLine(t.promptOverride.Get(), raw))
		fn(t, raw)
		return
	}

	cmd := strings.TrimSpace(raw)
	if cmd == "" {
		return
	}
	if t.sh == nil {
		return
	}
	// The boot block is no longer the whole scrollback, so a late-arriving measurement must not
	// rewrite a line by index any more.
	t.bootLive.Set(false)
	t.sh.nav.record(cmd)
	fields := strings.Fields(cmd)
	name := fields[0]
	args := fields[1:]
	logCommand(name)

	if cmd == "clear" {
		t.scrollback.Set(nil)
		return
	}
	t.emit(echoLine(t.sh.prompt(), cmd))

	// A line containing a pipe, a redirect or a chain is a shell line even when it starts with a
	// portfolio program's name — that is what makes `projects | grep Go` work.
	if !strings.ContainsAny(cmd, "|>&") {
		if runSpecial(name, args, t) {
			return
		}
		if isPortfolio(name) {
			t.emitRows(programOutput(name, args))
			t.emit(gap())
			return
		}
	}
	for _, o := range t.sh.run(cmd) {
		t.emit(shellLine(o))
	}
	t.emit(gap())
}

// runSpecial handles the commands that are not a pure function of their arguments — they animate,
// call the backend, open a document, or take over the input line. It reports whether it handled
// the command.
func runSpecial(name string, args []string, t *termCtl) bool {
	switch name {
	case "tour":
		startTour(t)
	case "share":
		runShare(args, t)
	case "theme":
		runTheme(args, t)
	case "stats", "uptime":
		startStats(name, t)
	case "ping":
		startPing(t)
	case "bench":
		startBench(t)
	case "contact":
		startContact(t)
	case "resume":
		t.emitRows(resumeOut())
		t.emit(gap())
		openResume()
	default:
		return runEgg(name, args, t)
	}
	return true
}

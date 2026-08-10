//go:build js && wasm

package main

import (
	"syscall/js"
)

// `tour` exists because most visitors will never type anything. The terminal can be the best thing
// on the page and still be invisible to someone who does not know it is interactive, or who does
// not want to guess at a command. The tour types for them, at a speed that reads as typing rather
// than as a text dump, and stops the moment they take over.

// tourStep is one command the tour runs, with the line of context shown before it.
type tourStep struct {
	say string
	cmd string
}

// tourScript is the guided demo. It is short on purpose: four commands is a demo, twelve is a
// hostage situation. The order tells the story a recruiter needs — who, what, proof, how to reach.
var tourScript = []tourStep{
	{"Who I am:", "about"},
	{"What I've shipped:", "projects"},
	{"One of them, in detail:", "open cashflux"},
	{"This is not a mock-up — those numbers come off the server:", "stats"},
	{"And you can reach me without leaving this window:", "links"},
}

// typeDelayMs is the per-character delay while the tour "types". Fast enough not to test anyone's
// patience, slow enough to read as a human at a keyboard.
const typeDelayMs = 28

// stepPauseMs is the beat between a step's output finishing and the next one starting.
const stepPauseMs = 900

// startTour runs the guided demo. Under prefers-reduced-motion the typing animation is dropped
// entirely and every step's output is printed at once — the information is the point, and the
// animation is exactly the kind of motion that setting exists to switch off.
func startTour(t *termCtl) {
	tok := t.token()
	t.emitRows([]progRow{
		head("guided tour"),
		dim("type anything to take over — this stops immediately."),
	})
	t.emit(gap())

	if reducedMotion() {
		for _, s := range tourScript {
			t.emitRows([]progRow{dim(s.say)})
			runTourCommand(s.cmd, t)
		}
		t.emitRows([]progRow{dim("tour complete — the prompt is yours.")})
		t.emit(gap())
		return
	}
	runTourStep(t, tok, 0)
}

// runTourStep plays one step, then schedules the next. It re-checks the generation token at every
// stage: if the visitor typed a command, or hit Ctrl+C, the tour is over and must not append
// another line into someone else's session.
func runTourStep(t *termCtl, tok, i int) {
	if !t.live(tok) {
		return
	}
	if i >= len(tourScript) {
		t.emitRows([]progRow{dim("tour complete — the prompt is yours.")})
		t.emit(gap())
		return
	}
	step := tourScript[i]
	t.emitRows([]progRow{dim(step.say)})
	typeInto(t, tok, step.cmd, func() {
		if !t.live(tok) {
			return
		}
		t.input.Set("")
		runTourCommand(step.cmd, t)
		after(stepPauseMs, func() { runTourStep(t, tok, i+1) })
	})
}

// runTourCommand runs one scripted command and prints it the way a typed command is printed. It
// deliberately does NOT go through runCommand: that bumps the generation counter, which would
// cancel the very tour that called it.
func runTourCommand(cmd string, t *termCtl) {
	if t.sh == nil {
		return
	}
	t.emit(echoLine(t.sh.prompt(), cmd))
	name, args := parseCmd(cmd)
	// The NAME, not the line. These are scripted commands with no visitor data in them, and the
	// allowlist would drop a whole line anyway — but telemetry.go claims that no code path can send
	// anything but a bare token, and a claim that depends on a downstream filter to stay true is
	// not the claim that was made.
	logCommand(name)
	if runSpecial(name, args, t) {
		return
	}
	if isPortfolio(name) {
		t.emitRows(programOutput(name, args))
		t.emit(gap())
		return
	}
	for _, o := range t.sh.run(cmd) {
		t.emit(shellLine(o))
	}
	t.emit(gap())
}

// typeInto animates text into the input a character at a time, then calls done.
func typeInto(t *termCtl, tok int, text string, done func()) {
	var step func(i int)
	step = func(i int) {
		if !t.live(tok) {
			return
		}
		if i > len(text) {
			done()
			return
		}
		t.input.Set(text[:i])
		after(typeDelayMs, func() { step(i + 1) })
	}
	step(0)
}

// reducedMotion reports whether the visitor asked for reduced motion.
func reducedMotion() bool {
	mq := js.Global().Call("matchMedia", "(prefers-reduced-motion: reduce)")
	return mq.Truthy() && mq.Get("matches").Bool()
}

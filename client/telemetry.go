//go:build js && wasm

package main

import (
	"context"
	"time"

	"github.com/monstercameron/earlcameron/internal/system"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// Command telemetry answers one question Cam cannot otherwise answer: does anyone type anything
// into the terminal, and if so, what? Without it, every future decision about this terminal is a
// guess.
//
// It is built to collect the least data that answers that question:
//
//   - Only the command NAME is sent — never arguments, never the raw line. `open cashflux` is sent
//     as "open". A visitor's typing is not transmitted even in part.
//   - The name is checked against the shipped command list on the CLIENT before it is sent, and
//     again on the SERVER before it is stored. An unrecognised word — which is what a typo, a
//     password pasted into the wrong window, or anything personal would look like — is dropped at
//     both ends rather than recorded.
//   - Nothing identifies the visitor: no id, no session, no IP, no user agent. The server stores a
//     day, a name, and a count.
//   - It is fire-and-forget. A failure is never surfaced, never retried, and never delays output.
//
// Cam's outstanding decision on whether to keep this at all is tracked in TODOS §15.F. Removing it
// is this file, the RecordCommand RPC, and the terminal_commands table — nothing else depends on it.

// logCommand reports that a command ran, if it is one this terminal ships.
//
// It takes the raw first token rather than the whole line precisely so there is no code path in
// which arguments could be sent: there is nothing here to strip, because nothing else is passed in.
func logCommand(name string) {
	if !system.KnownCommands[name] {
		return
	}
	go func() {
		cl, err := systemClient()
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = cl.RecordCommand(ctx, &sitepb.CommandEvent{Name: name})
	}()
}

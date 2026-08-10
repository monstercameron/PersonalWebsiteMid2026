//go:build js && wasm

package main

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// `contact` reads a message a field at a time and sends it over the same gRPC tunnel everything
// else uses. A recruiter who has got this far should not have to leave the terminal, open a mail
// client, and retype an address to say one sentence — the friction is where the intent dies.
//
// The form reuses the terminal's own input line rather than rendering a second one: the prompt
// changes to the field being asked for, and the submitted line is routed to the form instead of to
// the command runner. Ctrl+C abandons it, like any other prompt.

// contactDraft accumulates the fields as they are answered.
type contactDraft struct {
	name  string
	email string
}

// startContact begins the interactive message form.
func startContact(t *termCtl) {
	t.emitRows([]progRow{
		head("contact — send me a message from here"),
		dim("three questions, then it goes straight to my inbox. Ctrl+C to abandon."),
	})
	askName(t, &contactDraft{})
}

// askName asks for the sender's name.
func askName(t *termCtl, d *contactDraft) {
	t.beginCapture("your name", func(t *termCtl, in string) {
		name := strings.TrimSpace(in)
		if name == "" {
			t.emitRows([]progRow{{text: "a name, or Ctrl+C to abandon", style: rowErr}})
			askName(t, d)
			return
		}
		d.name = name
		askEmail(t, d)
	})
}

// askEmail asks for a reply address, checked here only for obvious nonsense — the server validates
// it properly, and this check exists to save a round trip, not to replace that one.
func askEmail(t *termCtl, d *contactDraft) {
	t.beginCapture("your email", func(t *termCtl, in string) {
		email := strings.TrimSpace(in)
		if !strings.Contains(email, "@") || len(email) < 5 {
			t.emitRows([]progRow{{text: "that doesn't look like an email — try again, or Ctrl+C", style: rowErr}})
			askEmail(t, d)
			return
		}
		d.email = email
		askBody(t, d)
	})
}

// askBody asks for the message itself and sends it.
func askBody(t *termCtl, d *contactDraft) {
	t.beginCapture("message", func(t *termCtl, in string) {
		body := strings.TrimSpace(in)
		if body == "" {
			t.emitRows([]progRow{{text: "say something — or Ctrl+C to abandon", style: rowErr}})
			askBody(t, d)
			return
		}
		t.endCapture()
		sendContact(t, d.name, d.email, body)
	})
}

// sendContact posts the message over ContactService and reports what actually happened. A failure
// prints the direct email address, because the one outcome that must never occur is a visitor
// believing they have reached Cam when they have not.
func sendContact(t *termCtl, name, email, body string) {
	// Captured before the goroutine starts. Every path below re-checks it, including the failures:
	// this call can take seconds, and if the visitor has run something else meanwhile, replaceLast
	// would overwrite that command's last line and the result would be filed under the wrong
	// command. A send whose outcome cannot be shown is dropped rather than shown in the wrong place.
	tok := t.token()
	t.emitRows([]progRow{dim("sending…")})
	go func() {
		fail := func(msg string) {
			if !t.live(tok) {
				return
			}
			t.replaceLast(renderRow(progRow{text: msg, style: rowErr}))
			t.emitRows([]progRow{dim("email me directly instead: cam@earlcameron.com")})
			t.emit(gap())
		}
		cl, err := contactClient()
		if err != nil {
			fail("contact: tunnel unavailable")
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		ack, err := cl.SendMessage(ctx, &sitepb.ContactMessage{Name: name, Email: email, Body: body})
		if err != nil {
			fail("contact: " + grpcMessage(err))
			return
		}
		if !t.live(tok) {
			return
		}
		msg := ack.GetMessage()
		if msg == "" {
			msg = "Sent."
		}
		t.replaceLast(renderRow(row("✓ " + msg)))
		t.emitRows([]progRow{dim("sent as " + name + " <" + email + ">")})
		t.emit(gap())
	}()
}

// grpcMessage extracts the human-readable part of a gRPC error. The server's validation messages
// are written to be read by a person, and wrapping them in "rpc error: code = ..." would throw
// that away.
func grpcMessage(err error) string {
	s := err.Error()
	if i := strings.LastIndex(s, "desc = "); i >= 0 {
		return s[i+len("desc = "):]
	}
	return s
}

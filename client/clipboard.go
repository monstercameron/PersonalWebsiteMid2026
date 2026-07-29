//go:build js && wasm

package main

import "syscall/js"

// copyFeedbackMs is how long a copy button shows its result before returning to "copy".
const copyFeedbackMs = 1600

// bindCopyButtons wires every server-rendered `data-copy-url` button to the Clipboard API.
//
// The buttons are part of the SSR shell (internal/site.feedURLRow), so they exist in the DOM before
// this binary runs and cannot be given handlers at render time — the same situation as the hero's
// terminal CTA. One delegated listener on the document covers all of them and needs no cleanup: it
// lives as long as the page does.
//
// Copying can fail for reasons that are not bugs — navigator.clipboard is undefined outside a
// secure context, and the write can be rejected — so the button reports what happened instead of
// pretending. The URL also sits in a readonly field beside each button, which is the real fallback:
// if this never binds at all, the address is still there to select by hand.
func bindCopyButtons() {
	doc := js.Global().Get("document")
	doc.Call("addEventListener", "click", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		target := args[0].Get("target")
		if !target.Truthy() {
			return nil
		}
		// The click can land on the button or on a node inside it; closest() resolves both.
		btn := target.Call("closest", "[data-copy-url]")
		if !btn.Truthy() {
			return nil
		}
		url := btn.Call("getAttribute", "data-copy-url").String()
		if url == "" {
			return nil
		}
		writeClipboard(url, btn)
		return nil
	}))
}

// writeClipboard copies text and reports the outcome on the button that asked for it.
func writeClipboard(text string, btn js.Value) {
	clip := js.Global().Get("navigator").Get("clipboard")
	if !clip.Truthy() {
		// No Clipboard API — an insecure origin, typically. Point at the field rather than failing
		// silently, since the field is the way out.
		flashLabel(btn, "select it instead")
		return
	}
	promise := clip.Call("writeText", text)

	// then/catch handlers must be released or they leak for the life of the page. A promise settles
	// exactly once, so releasing both from whichever fires is safe.
	var onOK, onErr js.Func
	release := func() {
		onOK.Release()
		onErr.Release()
	}
	onOK = js.FuncOf(func(js.Value, []js.Value) any {
		flashLabel(btn, "copied")
		release()
		return nil
	})
	onErr = js.FuncOf(func(js.Value, []js.Value) any {
		flashLabel(btn, "copy failed")
		release()
		return nil
	})
	promise.Call("then", onOK).Call("catch", onErr)
}

// flashLabel swaps a button's text for a moment, then restores it. The original label is read at
// call time rather than assumed, so a second click mid-flash cannot leave the button stuck showing
// "copied" forever.
func flashLabel(btn js.Value, msg string) {
	if btn.Get("dataset").Get("flashing").Truthy() {
		return
	}
	original := btn.Get("textContent").String()
	btn.Get("dataset").Set("flashing", "1")
	btn.Set("textContent", msg)

	var restore js.Func
	restore = js.FuncOf(func(js.Value, []js.Value) any {
		btn.Set("textContent", original)
		btn.Get("dataset").Delete("flashing")
		restore.Release()
		return nil
	})
	js.Global().Call("setTimeout", restore, copyFeedbackMs)
}

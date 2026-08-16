//go:build js && wasm

package main

import "syscall/js"

// bindNavPopover auto-dismisses the nav's <details data-nav-popover> disclosures (the "Feeds"
// menu) when attention leaves them — a click anywhere else, or focus moving out (Tab past the
// last item, a click into a form field). Native <details> has no outside-dismiss of its own, so
// without this the menu sat open until its summary was clicked again.
//
// Same shape as bindCopyButtons: the element is SSR markup that exists before this binary runs,
// so a pair of delegated document listeners covers it with no cleanup needed — they live as long
// as the page does. If the wasm never boots, the popover still opens and closes on its summary;
// only the auto-dismiss is an enhancement.
//
// pointerdown rather than click, deliberately: closing on the down-stroke means a click that
// lands on another control dismisses the menu before that control acts, matching how every
// native menu behaves. focusin covers the keyboard path the pointer listener cannot see.
func bindNavPopover() {
	doc := js.Global().Get("document")
	closeUnless := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		target := args[0].Get("target")
		open := doc.Call("querySelectorAll", "details[data-nav-popover][open]")
		for i := 0; i < open.Length(); i++ {
			d := open.Index(i)
			// Element.contains(target) is false for a click on the page body and for focus in
			// any other control; the popover's own summary and items keep it open.
			if !target.Truthy() || !d.Call("contains", target).Bool() {
				d.Call("removeAttribute", "open")
			}
		}
		return nil
	})
	doc.Call("addEventListener", "pointerdown", closeUnless)
	doc.Call("addEventListener", "focusin", closeUnless)
}

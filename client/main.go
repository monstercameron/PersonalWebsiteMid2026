//go:build js && wasm

// Command client is the GoWebComponents (Go→WASM) frontend. On the public site it mounts the
// interactive terminal over the SSR shell; on /admin it mounts the admin console, which talks to
// the backend purely over gRPC (AdminService via the GoGRPCBridge WebSocket tunnel). The only
// JavaScript in the project is the wasm bootstrap that loads this binary. Build: GOOS=js GOARCH=wasm.
package main

import (
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// main mounts the right app for the current path: the admin console under /admin, else the terminal.
func main() {
	path := js.Global().Get("location").Get("pathname").String()
	if strings.HasPrefix(path, "/admin") {
		ui.Run("#admin-root", AdminApp)
		return
	}
	// Bound before the terminal mounts: the copy buttons belong to the SSR shell, not to any
	// component, so they are wired directly rather than through the reconciler.
	bindCopyButtons()
	bindNavPopover()
	ui.Run("#term-root", Terminal)
}

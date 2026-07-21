//go:build js && wasm

// Command client is the GoWebComponents (Go→WASM) frontend. It mounts the interactive terminal
// over the server-rendered standard site and (next) dials the backend via gRPC-over-WebSocket.
// The only JavaScript in the project is the wasm bootstrap that loads this binary. Build:
// GOOS=js GOARCH=wasm.
package main

import "github.com/monstercameron/GoWebComponents/v4/ui"

// main mounts the terminal into the #term-root element placed by the SSR shell.
func main() {
	ui.Run("#term-root", Terminal)
}

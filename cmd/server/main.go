// Command server is the ingress entrypoint for earlcameron.com.
//
// One process, one listener: it will serve both the gRPC-over-WebSocket tunnel (app data
// plane) and the document plane (wasm, SSR pages, RSS, PDFs). See internal/server.
package main

import (
	"log"

	"github.com/monstercameron/earlcameron/internal/config"
	"github.com/monstercameron/earlcameron/internal/server"
)

// main loads configuration and runs the ingress server until interrupted.
func main() {
	srv, err := server.New(config.Load())
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

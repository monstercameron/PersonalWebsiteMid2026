// Command server is the ingress entrypoint for earlcameron.com.
//
// One process, one listener: it will serve both the gRPC-over-WebSocket tunnel (app data
// plane) and the document plane (wasm, SSR pages, RSS, PDFs). See internal/server.
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/monstercameron/earlcameron/internal/config"
	"github.com/monstercameron/earlcameron/internal/server"
)

// main loads configuration and runs the ingress server until interrupted.
//
// `server healthcheck` is how the Docker container checks itself: the runtime image is
// distroless (no shell, no curl), so the binary is the only thing available to ask
// /healthz — the same endpoint deploy/update.sh's legacy flow gates on. Exit 0 = healthy.
func main() {
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheckCmd())
	}
	srv, err := server.New(config.Load())
	if err != nil {
		log.Fatal(err)
	}
	if err := srv.Run(); err != nil {
		log.Fatal(err)
	}
}

// healthcheckCmd GETs /healthz on this instance's own listen address and reports the
// answer as an exit code. The address comes from the same LISTEN_ADDR the server reads,
// so the check and the listener cannot drift; a bare ":8095" form is asked on loopback.
func healthcheckCmd() int {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8095"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.Status)
		return 1
	}
	return 0
}

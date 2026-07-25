//go:build js && wasm

package main

import (
	"context"
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// tokenKey is the localStorage key holding the admin JWT.
const tokenKey = "ec_admin_jwt"

// adminConn is the lazily-dialed gRPC-over-WebSocket connection to the backend (same-origin /socket).
var adminConn *grpc.ClientConn

// adminClient returns the AdminService client, dialing the tunnel on first use. This is the browser
// side of the data plane: the admin UI talks to the Go backend purely over gRPC, no HTTP forms.
func adminClient() (sitepb.AdminServiceClient, error) {
	if adminConn == nil {
		// The WebSocket tunnel carries transport security at the browser layer, but grpc-go still
		// requires credentials to be set explicitly, so pass insecure creds for the gRPC layer.
		conn, err := grpctunnel.BuildTunnelConn(context.Background(), grpctunnel.TunnelConfig{
			Target:      "/socket",
			GRPCOptions: []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())},
		})
		if err != nil {
			return nil, err
		}
		adminConn = conn
	}
	return sitepb.NewAdminServiceClient(adminConn), nil
}

// callCtx builds a per-call context with a timeout and the admin JWT in "authorization" metadata
// (the server interceptor requires it on every method but Login).
func callCtx(token string) (context.Context, context.CancelFunc) {
	return callCtxTimeout(token, 30*time.Second)
}

// callCtxLong is for methods that call OpenAI server-side (dry-run / generate-and-post): its deadline
// exceeds the server's OpenAI HTTP timeout so a slow generation surfaces its real result instead of a
// premature client timeout (which, on the post path, could otherwise prompt a retry and a duplicate
// Slack post).
func callCtxLong(token string) (context.Context, context.CancelFunc) {
	return callCtxTimeout(token, 75*time.Second)
}

// callCtxTimeout builds a per-call context with the given timeout and the admin JWT in
// "authorization" metadata.
func callCtxTimeout(token string, d time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}
	return ctx, cancel
}

// currentAdminView derives the active admin view from the URL path so /admin/resume, /admin/settings,
// and /admin (or /admin/anime) deep-link to the right view.
func currentAdminView() string {
	p := js.Global().Get("location").Get("pathname").String()
	switch {
	case strings.HasSuffix(p, "/resume"):
		return "resume"
	case strings.HasSuffix(p, "/settings"):
		return "settings"
	case strings.HasSuffix(p, "/rss"):
		return "rss"
	case strings.HasSuffix(p, "/cashflux"):
		return "cashflux"
	default:
		return "anime"
	}
}

// copyToClipboard writes text to the system clipboard via the async Clipboard API. Best-effort and
// fire-and-forget (the Promise result is ignored) — the common failure mode (an insecure context or
// a browser without the API) isn't worth surfacing as an error for a convenience action; the code is
// still fully visible and selectable on-screen either way.
func copyToClipboard(text string) {
	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return
	}
	clipboard := nav.Get("clipboard")
	if !clipboard.Truthy() || !clipboard.Get("writeText").Truthy() {
		return
	}
	clipboard.Call("writeText", text)
}

// pushAdminPath updates the browser URL to the sub-route for view without reloading (history API).
func pushAdminPath(view string) {
	js.Global().Get("history").Call("pushState", nil, "", "/admin/"+view)
}

// onPopState registers fn for browser back/forward navigation; the returned func unregisters it.
func onPopState(fn func()) func() {
	cb := js.FuncOf(func(js.Value, []js.Value) any { fn(); return nil })
	win := js.Global().Get("window")
	win.Call("addEventListener", "popstate", cb)
	return func() {
		win.Call("removeEventListener", "popstate", cb)
		cb.Release()
	}
}

// loadToken reads the stored admin JWT from localStorage.
func loadToken() string {
	v := js.Global().Get("localStorage").Call("getItem", tokenKey)
	if v.Truthy() {
		return v.String()
	}
	return ""
}

// saveToken persists the admin JWT to localStorage.
func saveToken(t string) { js.Global().Get("localStorage").Call("setItem", tokenKey, t) }

// clearToken removes the stored admin JWT (logout).
func clearToken() { js.Global().Get("localStorage").Call("removeItem", tokenKey) }

// openInNewTab navigates to url in a new tab with the opener relationship severed.
// "noopener" matters here because the URL can carry a single-use activation code:
// without it the opened page could reach back through window.opener, and a handoff
// credential is not something to leave reachable from another document.
func openInNewTab(url string) {
	js.Global().Call("open", url, "_blank", "noopener")
}

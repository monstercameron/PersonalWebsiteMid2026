//go:build js && wasm

package main

import (
	"context"
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if token != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}
	return ctx, cancel
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

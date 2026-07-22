package admin

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// adminMethodPrefix is the fully-qualified gRPC method prefix for AdminService.
const adminMethodPrefix = "/site.v1.AdminService/"

// publicMethods are the AdminService methods reachable without a session token: login and the
// first-run / lockout auth flows (the owner has no token in exactly these cases).
var publicMethods = map[string]bool{
	"/site.v1.AdminService/Login":         true,
	"/site.v1.AdminService/AuthState":     true,
	"/site.v1.AdminService/Setup":         true,
	"/site.v1.AdminService/ResetPassword": true,
}

// UnaryAuthInterceptor returns a gRPC unary interceptor that enforces the admin session: every
// AdminService method except Login requires a valid token in the "authorization" metadata
// ("Bearer <token>"). Calls to other services (ContentService, ContactService) pass through
// untouched, so the public data plane is unaffected.
func (s *Sessions) UnaryAuthInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !strings.HasPrefix(info.FullMethod, adminMethodPrefix) || publicMethods[info.FullMethod] {
			return handler(ctx, req)
		}
		if !s.tokenFromContext(ctx) {
			return nil, status.Error(codes.Unauthenticated, "admin session required")
		}
		return handler(ctx, req)
	}
}

// tokenFromContext reports whether ctx carries a valid admin token in "authorization" metadata.
func (s *Sessions) tokenFromContext(ctx context.Context) bool {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	for _, v := range md.Get("authorization") {
		if s.Verify(ctx, strings.TrimPrefix(v, "Bearer ")) {
			return true
		}
	}
	return false
}

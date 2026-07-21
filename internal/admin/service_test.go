package admin

import (
	"context"
	"net"
	"path/filepath"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// TestAdminServiceAuthPlane exercises the gRPC admin data plane over a real client/server pair:
// login (wrong + right password), the interceptor rejecting unauthenticated calls, an authed call
// succeeding, and the disabled-tailoring precondition. No external network is touched.
func TestAdminServiceAuthPlane(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer st.Close()

	sessions := NewSessions("cam", "secret-pw", "test-secret")
	// empty OpenAI key → tailoring disabled
	svc := NewService(anime.New(st), sessions, st, func(context.Context) (string, string) { return "", "gpt-4o-mini" })

	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer(grpc.UnaryInterceptor(sessions.UnaryAuthInterceptor()))
	sitepb.RegisterAdminServiceServer(srv, svc)
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()
	client := sitepb.NewAdminServiceClient(conn)
	ctx := context.Background()

	// Wrong password is rejected.
	if r, err := client.Login(ctx, &sitepb.LoginRequest{Username: "cam", Password: "nope"}); err != nil || r.GetOk() {
		t.Fatalf("wrong password should fail: ok=%v err=%v", r.GetOk(), err)
	}
	// Wrong username is rejected too.
	if r, err := client.Login(ctx, &sitepb.LoginRequest{Username: "eve", Password: "secret-pw"}); err != nil || r.GetOk() {
		t.Fatalf("wrong username should fail: ok=%v err=%v", r.GetOk(), err)
	}

	// Correct credentials mint a token.
	login, err := client.Login(ctx, &sitepb.LoginRequest{Username: "cam", Password: "secret-pw"})
	if err != nil || !login.GetOk() || login.GetToken() == "" {
		t.Fatalf("login should succeed with a token: %+v err=%v", login, err)
	}

	// An authenticated method without a token is rejected by the interceptor.
	if _, err := client.GetResume(ctx, &sitepb.Empty{}); status.Code(err) != codes.Unauthenticated {
		t.Fatalf("GetResume without token: want Unauthenticated, got %v", err)
	}

	// With the token it succeeds.
	authCtx := metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+login.GetToken())
	res, err := client.GetResume(authCtx, &sitepb.Empty{})
	if err != nil || res.GetName() == "" || len(res.GetJobs()) == 0 {
		t.Fatalf("GetResume with token should return the résumé: %v", err)
	}

	// Tailoring is authenticated but disabled without an OpenAI key.
	if _, err := client.TailorResume(authCtx, &sitepb.TailorRequest{JobUrl: "https://example.com/job"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("TailorResume without key: want FailedPrecondition, got %v", err)
	}
}

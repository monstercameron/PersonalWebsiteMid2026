package admin

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/openai"
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

	// No stored account: falls back to the env bootstrap credentials.
	sessions := NewSessions(st, "cam", "secret-pw", "test-secret", "", "")
	// empty OpenAI key → tailoring disabled
	svc := NewService(anime.New(st), sessions, st, "https://example.com", func(context.Context) (string, string) { return "", "gpt-4o-mini" })

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

// TestPostScheduledIfDue covers the daily Slack scheduler's gating: disabled, before the post hour,
// the once-per-day guard, and that the day's slot is claimed even when the actual post can't proceed.
func TestPostScheduledIfDue(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer st.Close()
	sessions := NewSessions(st, "cam", "pw12345678", "sekret", "", "")
	// No OpenAI key and no webhook: publishDiscussion will fail, but the gating is what we test.
	svc := NewService(anime.New(st), sessions, st, "https://example.com", func(context.Context) (string, string) { return "", "" })
	ctx := context.Background()
	at := func(h int) time.Time { return time.Date(2026, 7, 22, h, 0, 0, 0, time.Local) }

	// Disabled → never posts.
	if _, posted, err := svc.PostScheduledIfDue(ctx, at(10)); posted || err != nil {
		t.Fatalf("disabled must not post: posted=%v err=%v", posted, err)
	}
	_ = st.SetSetting(ctx, store.SettingSlackEnabled, "1")
	_ = st.SetSetting(ctx, store.SettingSlackPostHour, "9")

	// Enabled but before the post hour → no post.
	if _, posted, err := svc.PostScheduledIfDue(ctx, at(8)); posted || err != nil {
		t.Fatalf("before hour must not post: posted=%v err=%v", posted, err)
	}
	// Hour reached, but no OpenAI key → generation fails; the day's slot is still claimed so a
	// broken config can't retry every tick. (A missing webhook alone no longer blocks publishing —
	// the RSS post is recorded and Slack is skipped.)
	_, posted, err := svc.PostScheduledIfDue(ctx, at(10))
	if posted {
		t.Fatal("no OpenAI key: should not report a successful post")
	}
	if err == nil {
		t.Fatal("expected an error when generation cannot run (no OpenAI key)")
	}
	// The slot is now claimed for the day, so a second tick the same day is a clean no-op (no retry).
	if _, posted, err := svc.PostScheduledIfDue(ctx, at(11)); posted || err != nil {
		t.Fatalf("already-posted day must no-op: posted=%v err=%v", posted, err)
	}
}

// TestPublishDiscussionRecordsAndDecouplesSlack proves the changed contract end-to-end (with the
// OpenAI call stubbed via openai.BaseURL): a publish RECORDS the post in the qotd_posts history
// even when (a) no Slack webhook is configured and (b) the Slack POST fails — Slack is best-effort
// delivery, and each outcome is stated in the returned message.
func TestPublishDiscussionRecordsAndDecouplesSlack(t *testing.T) {
	fakeOpenAI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"What arc should have ended sooner?"}`))
	}))
	defer fakeOpenAI.Close()
	oldBase := openai.BaseURL
	openai.BaseURL = fakeOpenAI.URL
	defer func() { openai.BaseURL = oldBase }()

	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer st.Close()
	ctx := context.Background()
	_ = st.SetSetting(ctx, store.SettingOpenAIKey, "test-key")
	sessions := NewSessions(st, "cam", "pw12345678", "sekret", "", "")
	svc := NewService(anime.New(st), sessions, st, "https://example.com", func(ctx context.Context) (string, string) {
		k, _ := st.GetSetting(ctx, store.SettingOpenAIKey)
		return k, "test-model"
	})

	// (a) No webhook: the RSS post is still recorded; the message says Slack was skipped.
	msg, err := svc.publishDiscussion(ctx)
	if err != nil {
		t.Fatalf("publish without webhook must succeed: %v", err)
	}
	if !strings.Contains(msg, "published to the RSS feed") || !strings.Contains(msg, "no Slack webhook") {
		t.Fatalf("message must state RSS publish + Slack skip, got %q", msg)
	}
	posts, err := st.RecentQOTDPosts(ctx, 10)
	if err != nil || len(posts) != 1 {
		t.Fatalf("want 1 recorded post, got %d (err %v)", len(posts), err)
	}
	if posts[0].Body != "What arc should have ended sooner?" {
		t.Fatalf("recorded body = %q", posts[0].Body)
	}

	// (b) Failing webhook: still recorded, message reports the Slack failure.
	deadSlack := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer deadSlack.Close()
	_ = st.SetSetting(ctx, store.SettingSlackWebhook, deadSlack.URL)
	msg, err = svc.publishDiscussion(ctx)
	if err != nil {
		t.Fatalf("publish with failing webhook must still succeed: %v", err)
	}
	if !strings.Contains(msg, "published to the RSS feed") || !strings.Contains(msg, "Slack post failed") {
		t.Fatalf("message must state RSS publish + Slack failure, got %q", msg)
	}
	if posts, _ := st.RecentQOTDPosts(ctx, 10); len(posts) != 2 {
		t.Fatalf("want 2 recorded posts after second publish, got %d", len(posts))
	}
}

// TestAuthThrottleNotBypassable proves the credential-check delay can't be skipped by cancelling the
// context (the timing-oracle / throttle-bypass a cancellable delay would allow): a failed Login on an
// already-cancelled context still takes at least the uniform authFloor.
func TestAuthThrottleNotBypassable(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer st.Close()
	sessions := NewSessions(st, "cam", "envpassword", "sekret", "", "")
	svc := NewService(anime.New(st), sessions, st, "https://example.com", func(context.Context) (string, string) { return "", "" })

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled: a cancellable throttle would return immediately
	start := time.Now()
	rep, _ := svc.Login(ctx, &sitepb.LoginRequest{Username: "cam", Password: "wrong"})
	elapsed := time.Since(start)
	if rep.GetOk() {
		t.Fatal("a wrong password must fail")
	}
	if elapsed < authFloor-100*time.Millisecond {
		t.Fatalf("throttle bypassed via context cancel: Login returned in %v, want >= %v", elapsed, authFloor)
	}
}

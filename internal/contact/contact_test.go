package contact

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// newTestService returns a Service backed by a throwaway on-disk store.
func newTestService(t *testing.T) *Service {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return New(st)
}

// TestSendMessageValid stores a well-formed message.
func TestSendMessageValid(t *testing.T) {
	ack, err := newTestService(t).SendMessage(context.Background(),
		&sitepb.ContactMessage{Name: "Ada", Email: "ada@example.com", Body: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if !ack.GetOk() {
		t.Error("ack not ok")
	}
}

// TestSendMessageInvalid rejects missing fields and malformed email with InvalidArgument.
func TestSendMessageInvalid(t *testing.T) {
	svc := newTestService(t)
	cases := []*sitepb.ContactMessage{
		{Name: "", Email: "a@b.com", Body: "x"},
		{Name: "A", Email: "no-at-sign", Body: "x"},
		{Name: "A", Email: "a@b.com", Body: ""},
	}
	for _, c := range cases {
		if _, err := svc.SendMessage(context.Background(), c); status.Code(err) != codes.InvalidArgument {
			t.Errorf("want InvalidArgument for %+v, got %v", c, err)
		}
	}
}

// TestRateLimitPerMinute caps a burst. The terminal's `contact` command puts a send button in
// front of every visitor, so an unbounded write path into the database is not acceptable.
func TestRateLimitPerMinute(t *testing.T) {
	svc := newTestService(t)
	msg := func() *sitepb.ContactMessage {
		return &sitepb.ContactMessage{Name: "Ada", Email: "ada@example.com", Body: "hello"}
	}
	for i := 0; i < maxPerMinute; i++ {
		if _, err := svc.SendMessage(context.Background(), msg()); err != nil {
			t.Fatalf("message %d should have been accepted: %v", i+1, err)
		}
	}
	_, err := svc.SendMessage(context.Background(), msg())
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("got %v; want ResourceExhausted once the burst limit is hit", err)
	}
}

// TestRateLimitWindowExpires proves the limiter is a sliding window and not a permanent lockout —
// a visitor who arrives after a burst must still be able to write.
func TestRateLimitWindowExpires(t *testing.T) {
	svc := newTestService(t)
	old := time.Now().Add(-2 * time.Minute)
	for i := 0; i < maxPerMinute; i++ {
		if !svc.allow(old) {
			t.Fatalf("historical send %d should have been allowed", i+1)
		}
	}
	if !svc.allow(time.Now()) {
		t.Fatal("a send should be allowed once the earlier burst has aged out of the minute window")
	}
}

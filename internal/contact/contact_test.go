package contact

import (
	"context"
	"path/filepath"
	"testing"

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

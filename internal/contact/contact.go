// Package contact implements ContactService — validating and storing visitor messages
// server-side over the gRPC tunnel.
package contact

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// maxBodyLen caps a message body to keep abusive payloads out of the database.
const maxBodyLen = 5000

// Rate limits. These are process-global rather than per-visitor, deliberately: the RPC arrives
// over the WebSocket tunnel, where the peer address is the proxy's rather than the sender's, so a
// per-visitor limit here would be a limit on nginx and would lock out every legitimate visitor the
// moment one person got noisy. A global cap cannot stop a determined flooder, but it does bound
// how much junk can reach the database in an hour, which is what this is for.
//
// The terminal's `contact` command did not create this exposure — ContactService was already
// reachable over the tunnel — but it does advertise it, which is reason enough to bound it.
const (
	maxPerMinute = 5
	maxPerHour   = 40
)

// Service implements sitepb.ContactServiceServer, persisting messages via the store.
type Service struct {
	sitepb.UnimplementedContactServiceServer
	store *store.Store

	// mu guards the send-time ring used for rate limiting.
	mu    sync.Mutex
	sends []time.Time
}

// New returns a Service backed by the given store.
func New(s *store.Store) *Service { return &Service{store: s} }

// allow reports whether another message may be accepted now, recording it when it may. Timestamps
// older than an hour are dropped on each call, so the slice stays bounded by maxPerHour.
func (s *Service) allow(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := s.sends[:0]
	minuteAgo, hourAgo := now.Add(-time.Minute), now.Add(-time.Hour)
	inLastMinute := 0
	for _, t := range s.sends {
		if t.After(hourAgo) {
			kept = append(kept, t)
			if t.After(minuteAgo) {
				inLastMinute++
			}
		}
	}
	s.sends = kept
	if inLastMinute >= maxPerMinute || len(s.sends) >= maxPerHour {
		return false
	}
	s.sends = append(s.sends, now)
	return true
}

// SendMessage validates an inbound message and stores it, returning a friendly acknowledgement.
// Validation errors are returned as InvalidArgument; storage failures as Internal.
func (s *Service) SendMessage(ctx context.Context, m *sitepb.ContactMessage) (*sitepb.Ack, error) {
	name := strings.TrimSpace(m.GetName())
	email := strings.TrimSpace(m.GetEmail())
	body := strings.TrimSpace(m.GetBody())
	switch {
	case name == "" || email == "" || body == "":
		return nil, status.Error(codes.InvalidArgument, "name, email, and message are all required")
	case !strings.Contains(email, "@"):
		return nil, status.Error(codes.InvalidArgument, "that email doesn't look right")
	case len(body) > maxBodyLen:
		return nil, status.Error(codes.InvalidArgument, "that message is a bit too long")
	}
	if !s.allow(time.Now()) {
		return nil, status.Error(codes.ResourceExhausted,
			"too many messages just came through — try again in a minute, or email cam@earlcameron.com")
	}
	if err := s.store.SaveContact(ctx, store.ContactMessage{
		Name: name, Email: email, Body: body, CreatedAt: time.Now().Unix(),
	}); err != nil {
		return nil, status.Error(codes.Internal, "couldn't save your message — please try again")
	}
	return &sitepb.Ack{Ok: true, Message: "Thanks — I'll get back to you."}, nil
}

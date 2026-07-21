// Package contact implements ContactService — validating and storing visitor messages
// server-side over the gRPC tunnel.
package contact

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// maxBodyLen caps a message body to keep abusive payloads out of the database.
const maxBodyLen = 5000

// Service implements sitepb.ContactServiceServer, persisting messages via the store.
type Service struct {
	sitepb.UnimplementedContactServiceServer
	store *store.Store
}

// New returns a Service backed by the given store.
func New(s *store.Store) *Service { return &Service{store: s} }

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
	if err := s.store.SaveContact(ctx, store.ContactMessage{
		Name: name, Email: email, Body: body, CreatedAt: time.Now().Unix(),
	}); err != nil {
		return nil, status.Error(codes.Internal, "couldn't save your message — please try again")
	}
	return &sitepb.Ack{Ok: true, Message: "Thanks — I'll get back to you."}, nil
}

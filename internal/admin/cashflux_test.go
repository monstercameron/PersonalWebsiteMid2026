package admin

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cashfluxembed "github.com/monstercameron/CashFlux/pkg/embed"
	"github.com/monstercameron/earlcameron/internal/anime"
	"github.com/monstercameron/earlcameron/internal/store"
	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// fakeCashFluxAdmin is a CashFluxAdmin test double — no real embedded CashFlux store is needed to
// test Service's RPC mapping and error handling.
type fakeCashFluxAdmin struct {
	clients     []cashfluxembed.PhoneClient
	clientsErr  error
	mintCode    string
	mintExpires time.Time
	mintErr     error
	codes       []cashfluxembed.InviteCode
	codesErr    error
}

func (f *fakeCashFluxAdmin) ListClients() ([]cashfluxembed.PhoneClient, error) {
	return f.clients, f.clientsErr
}

func (f *fakeCashFluxAdmin) MintInviteCode() (string, time.Time, error) {
	return f.mintCode, f.mintExpires, f.mintErr
}

func (f *fakeCashFluxAdmin) ListInviteCodes() ([]cashfluxembed.InviteCode, error) {
	return f.codes, f.codesErr
}

func newCashFluxTestService(t *testing.T, cf CashFluxAdmin) *Service {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	sessions := NewSessions(st, "cam", "secret-pw", "test-secret", "", "")
	return NewService(anime.New(st), sessions, st, "https://example.com", func(context.Context) (string, string) { return "", "" }, cf)
}

func TestCashFluxRPCsFailPreconditionWhenNotConfigured(t *testing.T) {
	svc := newCashFluxTestService(t, nil)
	ctx := context.Background()

	if _, err := svc.ListCashFluxClients(ctx, &sitepb.Empty{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("ListCashFluxClients: err = %v, want FailedPrecondition", err)
	}
	if _, err := svc.MintCashFluxInviteCode(ctx, &sitepb.Empty{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("MintCashFluxInviteCode: err = %v, want FailedPrecondition", err)
	}
	if _, err := svc.ListCashFluxInviteCodes(ctx, &sitepb.Empty{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("ListCashFluxInviteCodes: err = %v, want FailedPrecondition", err)
	}
}

func TestListCashFluxClientsMapsFields(t *testing.T) {
	created := time.Date(2026, time.July, 20, 10, 0, 0, 0, time.UTC)
	verified := time.Date(2026, time.July, 20, 10, 5, 0, 0, time.UTC)
	fake := &fakeCashFluxAdmin{clients: []cashfluxembed.PhoneClient{
		{ID: "u1", PhoneNumber: "+15551234567", CreatedAt: created, PhoneVerifiedAt: verified},
	}}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.ListCashFluxClients(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("ListCashFluxClients: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("items = %d, want 1", len(resp.GetItems()))
	}
	item := resp.GetItems()[0]
	if item.GetPhoneNumber() != "+15551234567" {
		t.Fatalf("PhoneNumber = %q, want +15551234567", item.GetPhoneNumber())
	}
	if item.GetCreatedAt() != created.Unix() {
		t.Fatalf("CreatedAt = %d, want %d", item.GetCreatedAt(), created.Unix())
	}
	if item.GetPhoneVerifiedAt() != verified.Unix() {
		t.Fatalf("PhoneVerifiedAt = %d, want %d", item.GetPhoneVerifiedAt(), verified.Unix())
	}
}

func TestListCashFluxClientsPropagatesStoreError(t *testing.T) {
	fake := &fakeCashFluxAdmin{clientsErr: errors.New("db exploded")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.ListCashFluxClients(context.Background(), &sitepb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

func TestMintCashFluxInviteCodeMapsFields(t *testing.T) {
	expires := time.Date(2026, time.July, 20, 10, 15, 0, 0, time.UTC)
	fake := &fakeCashFluxAdmin{mintCode: "482913", mintExpires: expires}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.MintCashFluxInviteCode(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("MintCashFluxInviteCode: %v", err)
	}
	if resp.GetCode() != "482913" {
		t.Fatalf("Code = %q, want 482913", resp.GetCode())
	}
	if resp.GetExpiresAt() != expires.Unix() {
		t.Fatalf("ExpiresAt = %d, want %d", resp.GetExpiresAt(), expires.Unix())
	}
}

func TestListCashFluxInviteCodesReportsOutstandingVsConsumed(t *testing.T) {
	created := time.Date(2026, time.July, 20, 10, 0, 0, 0, time.UTC)
	expires := created.Add(15 * time.Minute)
	consumed := created.Add(5 * time.Minute)
	fake := &fakeCashFluxAdmin{codes: []cashfluxembed.InviteCode{
		{Code: "111111", CreatedAt: created, ExpiresAt: expires},                       // outstanding
		{Code: "222222", CreatedAt: created, ExpiresAt: expires, ConsumedAt: consumed}, // consumed
	}}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.ListCashFluxInviteCodes(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("ListCashFluxInviteCodes: %v", err)
	}
	items := resp.GetItems()
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].GetConsumedAt() != 0 {
		t.Fatalf("outstanding code ConsumedAt = %d, want 0", items[0].GetConsumedAt())
	}
	if items[1].GetConsumedAt() != consumed.Unix() {
		t.Fatalf("consumed code ConsumedAt = %d, want %d", items[1].GetConsumedAt(), consumed.Unix())
	}
}

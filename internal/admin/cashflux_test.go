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
	devices    []cashfluxembed.PendingDevice
	devicesErr error

	approveApproved bool
	approveCode     string
	approveErr      error
	approveDeviceID string // records the deviceID the last ApprovePairing call was made with

	rejectRejected bool
	rejectErr      error
	rejectDeviceID string // records the deviceID the last RejectPairing call was made with
}

func (f *fakeCashFluxAdmin) ListPendingDevices() ([]cashfluxembed.PendingDevice, error) {
	return f.devices, f.devicesErr
}

func (f *fakeCashFluxAdmin) ApprovePairing(deviceID string) (bool, string, error) {
	f.approveDeviceID = deviceID
	return f.approveApproved, f.approveCode, f.approveErr
}

func (f *fakeCashFluxAdmin) RejectPairing(deviceID string) (bool, error) {
	f.rejectDeviceID = deviceID
	return f.rejectRejected, f.rejectErr
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

	if _, err := svc.ListCashFluxPendingDevices(ctx, &sitepb.Empty{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("ListCashFluxPendingDevices: err = %v, want FailedPrecondition", err)
	}
	if _, err := svc.ApproveCashFluxPairing(ctx, &sitepb.CashFluxApprovePairingRequest{DeviceId: "d1"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("ApproveCashFluxPairing: err = %v, want FailedPrecondition", err)
	}
	if _, err := svc.RejectCashFluxPairing(ctx, &sitepb.CashFluxRejectPairingRequest{DeviceId: "d1"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("RejectCashFluxPairing: err = %v, want FailedPrecondition", err)
	}
}

func TestListCashFluxPendingDevicesMapsFields(t *testing.T) {
	requested := time.Date(2026, time.July, 20, 10, 0, 0, 0, time.UTC)
	expires := requested.Add(15 * time.Minute)
	fake := &fakeCashFluxAdmin{devices: []cashfluxembed.PendingDevice{
		{DeviceID: "dev-1", Label: "kitchen-tablet", RequestedAt: requested, ExpiresAt: expires},
	}}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.ListCashFluxPendingDevices(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("ListCashFluxPendingDevices: %v", err)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("items = %d, want 1", len(resp.GetItems()))
	}
	item := resp.GetItems()[0]
	if item.GetDeviceId() != "dev-1" {
		t.Fatalf("DeviceId = %q, want dev-1", item.GetDeviceId())
	}
	if item.GetLabel() != "kitchen-tablet" {
		t.Fatalf("Label = %q, want kitchen-tablet", item.GetLabel())
	}
	if item.GetRequestedAt() != requested.Unix() {
		t.Fatalf("RequestedAt = %d, want %d", item.GetRequestedAt(), requested.Unix())
	}
	if item.GetExpiresAt() != expires.Unix() {
		t.Fatalf("ExpiresAt = %d, want %d", item.GetExpiresAt(), expires.Unix())
	}
}

func TestListCashFluxPendingDevicesPropagatesStoreError(t *testing.T) {
	fake := &fakeCashFluxAdmin{devicesErr: errors.New("db exploded")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.ListCashFluxPendingDevices(context.Background(), &sitepb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

func TestApproveCashFluxPairingMapsFieldsAndPassesDeviceID(t *testing.T) {
	fake := &fakeCashFluxAdmin{approveApproved: true, approveCode: "482913"}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.ApproveCashFluxPairing(context.Background(), &sitepb.CashFluxApprovePairingRequest{DeviceId: "dev-1"})
	if err != nil {
		t.Fatalf("ApproveCashFluxPairing: %v", err)
	}
	if !resp.GetApproved() {
		t.Fatal("Approved = false, want true")
	}
	if resp.GetPairingCode() != "482913" {
		t.Fatalf("PairingCode = %q, want 482913", resp.GetPairingCode())
	}
	if fake.approveDeviceID != "dev-1" {
		t.Fatalf("ApprovePairing called with deviceID = %q, want dev-1", fake.approveDeviceID)
	}
}

// TestApproveCashFluxPairingAlreadyResolvedIsNotAnError proves the "already resolved" case
// (approved=false, no error, per pkg/embed.Admin.ApprovePairing's contract) is surfaced as a normal
// response — not turned into a gRPC error — so the client can distinguish it from a real failure.
func TestApproveCashFluxPairingAlreadyResolvedIsNotAnError(t *testing.T) {
	fake := &fakeCashFluxAdmin{approveApproved: false, approveCode: ""}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.ApproveCashFluxPairing(context.Background(), &sitepb.CashFluxApprovePairingRequest{DeviceId: "dev-1"})
	if err != nil {
		t.Fatalf("ApproveCashFluxPairing: %v", err)
	}
	if resp.GetApproved() {
		t.Fatal("Approved = true, want false")
	}
	if resp.GetPairingCode() != "" {
		t.Fatalf("PairingCode = %q, want empty", resp.GetPairingCode())
	}
}

func TestApproveCashFluxPairingPropagatesError(t *testing.T) {
	fake := &fakeCashFluxAdmin{approveErr: errors.New("store exploded")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.ApproveCashFluxPairing(context.Background(), &sitepb.CashFluxApprovePairingRequest{DeviceId: "dev-1"}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

func TestRejectCashFluxPairingMapsFieldsAndPassesDeviceID(t *testing.T) {
	fake := &fakeCashFluxAdmin{rejectRejected: true}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.RejectCashFluxPairing(context.Background(), &sitepb.CashFluxRejectPairingRequest{DeviceId: "dev-1"})
	if err != nil {
		t.Fatalf("RejectCashFluxPairing: %v", err)
	}
	if !resp.GetRejected() {
		t.Fatal("Rejected = false, want true")
	}
	if fake.rejectDeviceID != "dev-1" {
		t.Fatalf("RejectPairing called with deviceID = %q, want dev-1", fake.rejectDeviceID)
	}
}

// TestRejectCashFluxPairingAlreadyResolvedIsNotAnError mirrors the approve case: an already-resolved
// request reports rejected=false with no error.
func TestRejectCashFluxPairingAlreadyResolvedIsNotAnError(t *testing.T) {
	fake := &fakeCashFluxAdmin{rejectRejected: false}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.RejectCashFluxPairing(context.Background(), &sitepb.CashFluxRejectPairingRequest{DeviceId: "dev-1"})
	if err != nil {
		t.Fatalf("RejectCashFluxPairing: %v", err)
	}
	if resp.GetRejected() {
		t.Fatal("Rejected = true, want false")
	}
}

func TestRejectCashFluxPairingPropagatesError(t *testing.T) {
	fake := &fakeCashFluxAdmin{rejectErr: errors.New("store exploded")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.RejectCashFluxPairing(context.Background(), &sitepb.CashFluxRejectPairingRequest{DeviceId: "dev-1"}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

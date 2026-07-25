package admin

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
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
	activationCode      string
	activationExpiresAt time.Time
	activationErr       error
	activationCalls     int // records how many times MintActivationCode was called

	devices    []cashfluxembed.PendingDevice
	devicesErr error

	approveApproved bool
	approveCode     string
	approveErr      error
	approveDeviceID string // records the deviceID the last ApprovePairing call was made with

	rejectRejected bool
	rejectErr      error
	rejectDeviceID string // records the deviceID the last RejectPairing call was made with

	users         []cashfluxembed.User
	usersErr      error
	usersLim      int // records the limit the last ListUsers call was made with
	usersOff      int // records the offset the last ListUsers call was made with
	dbBytes       int64
	blobBytes     int64
	snapshotBytes int64
	statsErr      error

	deleteDeleted bool
	deleteErr     error
	deleteUserID  string // records the id the last DeleteUser call was made with

	createdID      string
	createErr      error
	createUsername string
	createRole     string

	updateErr      error
	updateUserID   string
	updateUsername string
	updateRole     string

	suspendErr       error
	suspendUserID    string
	suspendSuspended bool

	resetErr    error
	resetUserID string

	forUserCode    string
	forUserExpires time.Time
	forUserErr     error
	forUserID      string

	codeValid bool
	codeErr   error
	codeSeen  string
}

func (f *fakeCashFluxAdmin) MintActivationCodeForUser(userID string) (string, time.Time, error) {
	f.forUserID = userID
	return f.forUserCode, f.forUserExpires, f.forUserErr
}

func (f *fakeCashFluxAdmin) CreateUser(username, role string) (string, error) {
	f.createUsername, f.createRole = username, role
	return f.createdID, f.createErr
}

func (f *fakeCashFluxAdmin) UpdateUser(userID, username, role string) error {
	f.updateUserID, f.updateUsername, f.updateRole = userID, username, role
	return f.updateErr
}

func (f *fakeCashFluxAdmin) SetSuspended(userID string, suspended bool) error {
	f.suspendUserID, f.suspendSuspended = userID, suspended
	return f.suspendErr
}

func (f *fakeCashFluxAdmin) ResetCredentials(userID string) error {
	f.resetUserID = userID
	return f.resetErr
}

func (f *fakeCashFluxAdmin) ActivationCodeIsValid(code string) (bool, error) {
	f.codeSeen = code
	return f.codeValid, f.codeErr
}

func (f *fakeCashFluxAdmin) MintActivationCode() (string, time.Time, error) {
	f.activationCalls++
	return f.activationCode, f.activationExpiresAt, f.activationErr
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

func (f *fakeCashFluxAdmin) ListUsers(limit, offset int) ([]cashfluxembed.User, error) {
	f.usersLim, f.usersOff = limit, offset
	return f.users, f.usersErr
}

func (f *fakeCashFluxAdmin) DeleteUser(userID string) (bool, error) {
	f.deleteUserID = userID
	return f.deleteDeleted, f.deleteErr
}

func (f *fakeCashFluxAdmin) StorageStats() (int64, int64, int64, error) {
	return f.dbBytes, f.blobBytes, f.snapshotBytes, f.statsErr
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
	if _, err := svc.ListCashFluxUsers(ctx, &sitepb.CashFluxListUsersRequest{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("ListCashFluxUsers: err = %v, want FailedPrecondition", err)
	}
	if _, err := svc.GetCashFluxStorageStats(ctx, &sitepb.Empty{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("GetCashFluxStorageStats: err = %v, want FailedPrecondition", err)
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

// TestListCashFluxUsersMapsFieldsAndPassesLimitOffset proves every field maps onto the wire DTO
// (including the created_at → unix-seconds conversion) and that the request's limit/offset reach
// CashFluxAdmin.ListUsers unchanged — the server does no paging logic of its own, it's a pure
// pass-through to pkg/embed.Admin.ListUsers (which owns the clamping).
func TestListCashFluxUsersMapsFieldsAndPassesLimitOffset(t *testing.T) {
	createdAt := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	fake := &fakeCashFluxAdmin{users: []cashfluxembed.User{
		{ID: "u1", Provider: "device", Email: "cam@example.com", CreatedAt: createdAt,
			SubscriptionPlan: "free", SubscriptionStatus: "active", RequestsThisMonth: 42},
	}}
	svc := newCashFluxTestService(t, fake)

	resp, err := svc.ListCashFluxUsers(context.Background(), &sitepb.CashFluxListUsersRequest{Limit: 25, Offset: 50})
	if err != nil {
		t.Fatalf("ListCashFluxUsers: %v", err)
	}
	if fake.usersLim != 25 || fake.usersOff != 50 {
		t.Fatalf("ListUsers called with limit=%d offset=%d, want 25/50", fake.usersLim, fake.usersOff)
	}
	if len(resp.GetItems()) != 1 {
		t.Fatalf("items = %d, want 1", len(resp.GetItems()))
	}
	item := resp.GetItems()[0]
	if item.GetId() != "u1" || item.GetProvider() != "device" || item.GetEmail() != "cam@example.com" {
		t.Fatalf("item = %+v, want id/provider/email u1/device/cam@example.com", item)
	}
	if item.GetCreatedAt() != createdAt.Unix() {
		t.Fatalf("CreatedAt = %d, want %d", item.GetCreatedAt(), createdAt.Unix())
	}
	if item.GetSubscriptionPlan() != "free" || item.GetSubscriptionStatus() != "active" {
		t.Fatalf("subscription = %s/%s, want free/active", item.GetSubscriptionPlan(), item.GetSubscriptionStatus())
	}
	if item.GetRequestsThisMonth() != 42 {
		t.Fatalf("RequestsThisMonth = %d, want 42", item.GetRequestsThisMonth())
	}
}

func TestListCashFluxUsersEmpty(t *testing.T) {
	svc := newCashFluxTestService(t, &fakeCashFluxAdmin{})
	resp, err := svc.ListCashFluxUsers(context.Background(), &sitepb.CashFluxListUsersRequest{})
	if err != nil {
		t.Fatalf("ListCashFluxUsers: %v", err)
	}
	if len(resp.GetItems()) != 0 {
		t.Fatalf("items = %d, want 0", len(resp.GetItems()))
	}
}

func TestListCashFluxUsersPropagatesError(t *testing.T) {
	fake := &fakeCashFluxAdmin{usersErr: errors.New("db exploded")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.ListCashFluxUsers(context.Background(), &sitepb.CashFluxListUsersRequest{}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

func TestGetCashFluxStorageStatsMapsFields(t *testing.T) {
	fake := &fakeCashFluxAdmin{dbBytes: 12345, blobBytes: 67890}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.GetCashFluxStorageStats(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("GetCashFluxStorageStats: %v", err)
	}
	if resp.GetDbBytes() != 12345 || resp.GetBlobBytes() != 67890 {
		t.Fatalf("stats = %d/%d, want 12345/67890", resp.GetDbBytes(), resp.GetBlobBytes())
	}
}

func TestGetCashFluxStorageStatsPropagatesError(t *testing.T) {
	fake := &fakeCashFluxAdmin{statsErr: errors.New("disk error")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.GetCashFluxStorageStats(context.Background(), &sitepb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

func TestMintCashFluxActivationCodeMapsFields(t *testing.T) {
	expires := time.Date(2026, 7, 24, 19, 30, 0, 0, time.UTC)
	fake := &fakeCashFluxAdmin{activationCode: "482915", activationExpiresAt: expires}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.MintCashFluxActivationCode(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("MintCashFluxActivationCode: %v", err)
	}
	if resp.GetCode() != "482915" {
		t.Fatalf("code = %q, want 482915", resp.GetCode())
	}
	if got, want := resp.GetExpiresAt(), expires.Format(time.RFC3339); got != want {
		t.Fatalf("expiresAt = %q, want %q", got, want)
	}
	if fake.activationCalls != 1 {
		t.Fatalf("MintActivationCode called %d times, want 1", fake.activationCalls)
	}
}

// TestMintCashFluxActivationCodeNormalizesExpiryToUTC proves the wire value is always UTC, so the
// admin console never renders an expiry in whatever zone the server process happens to run in.
func TestMintCashFluxActivationCodeNormalizesExpiryToUTC(t *testing.T) {
	expires := time.Date(2026, 7, 24, 19, 30, 0, 0, time.FixedZone("EDT", -4*60*60))
	fake := &fakeCashFluxAdmin{activationCode: "111111", activationExpiresAt: expires}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.MintCashFluxActivationCode(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("MintCashFluxActivationCode: %v", err)
	}
	if got, want := resp.GetExpiresAt(), "2026-07-24T23:30:00Z"; got != want {
		t.Fatalf("expiresAt = %q, want %q", got, want)
	}
}

func TestMintCashFluxActivationCodePropagatesError(t *testing.T) {
	fake := &fakeCashFluxAdmin{activationErr: errors.New("store closed")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.MintCashFluxActivationCode(context.Background(), &sitepb.Empty{}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

// TestMintCashFluxActivationCodeUnconfigured proves a deployment with no embedded CashFlux reports
// FailedPrecondition, the same as every other CashFlux RPC, rather than panicking on a nil handle.
func TestMintCashFluxActivationCodeUnconfigured(t *testing.T) {
	svc := newCashFluxTestService(t, nil)
	if _, err := svc.MintCashFluxActivationCode(context.Background(), &sitepb.Empty{}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("err = %v, want FailedPrecondition", err)
	}
}

func TestDeleteCashFluxUserPassesIDAndReportsDeleted(t *testing.T) {
	fake := &fakeCashFluxAdmin{deleteDeleted: true}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.DeleteCashFluxUser(context.Background(), &sitepb.CashFluxDeleteUserRequest{UserId: "device:ABC"})
	if err != nil {
		t.Fatalf("DeleteCashFluxUser: %v", err)
	}
	if !resp.GetDeleted() {
		t.Fatal("Deleted = false, want true")
	}
	if fake.deleteUserID != "device:ABC" {
		t.Fatalf("DeleteUser called with %q, want device:ABC", fake.deleteUserID)
	}
}

// TestDeleteCashFluxUserAlreadyGoneIsNotAnError proves the "no such account" case
// (deleted=false, no error) reaches the client as a normal response rather than a gRPC error —
// a double-submit or a stale listing is work already done, not a failure to report.
func TestDeleteCashFluxUserAlreadyGoneIsNotAnError(t *testing.T) {
	fake := &fakeCashFluxAdmin{deleteDeleted: false}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.DeleteCashFluxUser(context.Background(), &sitepb.CashFluxDeleteUserRequest{UserId: "device:gone"})
	if err != nil {
		t.Fatalf("DeleteCashFluxUser: %v", err)
	}
	if resp.GetDeleted() {
		t.Fatal("Deleted = true, want false")
	}
}

func TestDeleteCashFluxUserPropagatesError(t *testing.T) {
	fake := &fakeCashFluxAdmin{deleteErr: errors.New("store exploded")}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.DeleteCashFluxUser(context.Background(), &sitepb.CashFluxDeleteUserRequest{UserId: "device:ABC"}); status.Code(err) != codes.Internal {
		t.Fatalf("err = %v, want Internal", err)
	}
}

func TestDeleteCashFluxUserUnconfigured(t *testing.T) {
	svc := newCashFluxTestService(t, nil)
	if _, err := svc.DeleteCashFluxUser(context.Background(), &sitepb.CashFluxDeleteUserRequest{UserId: "device:ABC"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("err = %v, want FailedPrecondition", err)
	}
}

// TestListCashFluxUsersMarksOwner proves the console is told which row is the owner's own
// account, so a delete of YOUR data can be worded differently from removing an invited person.
func TestListCashFluxUsersMarksOwner(t *testing.T) {
	fake := &fakeCashFluxAdmin{users: []cashfluxembed.User{
		{ID: cashfluxembed.OwnerAccountID, Provider: "device"},
		{ID: "device:SOMEONEELSE", Provider: "device"},
	}}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.ListCashFluxUsers(context.Background(), &sitepb.CashFluxListUsersRequest{})
	if err != nil {
		t.Fatalf("ListCashFluxUsers: %v", err)
	}
	if len(resp.GetItems()) != 2 {
		t.Fatalf("items = %d, want 2", len(resp.GetItems()))
	}
	if !resp.GetItems()[0].GetIsOwner() {
		t.Fatalf("%s: IsOwner = false, want true", cashfluxembed.OwnerAccountID)
	}
	if resp.GetItems()[1].GetIsOwner() {
		t.Fatal("device:SOMEONEELSE: IsOwner = true, want false")
	}
}

// TestListCashFluxUsersReportsSyncFacts proves the three sync fields reach the console, and that a
// never-synced account sends 0 rather than time.Time{}.Unix() — which is a large NEGATIVE number
// and would render as a date in year 1 instead of "never synced".
func TestListCashFluxUsersReportsSyncFacts(t *testing.T) {
	synced := time.Date(2026, 7, 24, 20, 19, 0, 0, time.UTC)
	fake := &fakeCashFluxAdmin{users: []cashfluxembed.User{
		{ID: "device:A", Workspaces: 2, DatasetBytes: 17064, LastSyncedAt: synced},
		{ID: "device:B"}, // never synced
	}}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.ListCashFluxUsers(context.Background(), &sitepb.CashFluxListUsersRequest{})
	if err != nil {
		t.Fatalf("ListCashFluxUsers: %v", err)
	}
	a, b := resp.GetItems()[0], resp.GetItems()[1]
	if a.GetWorkspaces() != 2 || a.GetDatasetBytes() != 17064 || a.GetLastSyncedAt() != synced.Unix() {
		t.Fatalf("device:A = %d/%d/%d, want 2/17064/%d",
			a.GetWorkspaces(), a.GetDatasetBytes(), a.GetLastSyncedAt(), synced.Unix())
	}
	if b.GetLastSyncedAt() != 0 {
		t.Fatalf("never-synced LastSyncedAt = %d, want 0", b.GetLastSyncedAt())
	}
}

func TestGetCashFluxStorageStatsIncludesSnapshotBytes(t *testing.T) {
	fake := &fakeCashFluxAdmin{dbBytes: 290816, blobBytes: 0, snapshotBytes: 17064}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.GetCashFluxStorageStats(context.Background(), &sitepb.Empty{})
	if err != nil {
		t.Fatalf("GetCashFluxStorageStats: %v", err)
	}
	if resp.GetSnapshotBytes() != 17064 {
		t.Fatalf("SnapshotBytes = %d, want 17064", resp.GetSnapshotBytes())
	}
}

func TestCreateCashFluxUserPassesFields(t *testing.T) {
	fake := &fakeCashFluxAdmin{createdID: "device:NEW"}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.CreateCashFluxUser(context.Background(), &sitepb.CashFluxCreateUserRequest{Username: "priya", Role: "viewer"})
	if err != nil {
		t.Fatalf("CreateCashFluxUser: %v", err)
	}
	if resp.GetUserId() != "device:NEW" || fake.createUsername != "priya" || fake.createRole != "viewer" {
		t.Fatalf("got id=%q username=%q role=%q", resp.GetUserId(), fake.createUsername, fake.createRole)
	}
}

// TestUpdateCashFluxUserSurfacesRefusalReason proves a refusal (demoting the owner, an unknown
// role, a taken username) reaches the console as InvalidArgument carrying the reason, rather than
// a generic failure the operator has to guess at.
func TestUpdateCashFluxUserSurfacesRefusalReason(t *testing.T) {
	fake := &fakeCashFluxAdmin{updateErr: errors.New("the owner account cannot be demoted")}
	svc := newCashFluxTestService(t, fake)
	_, err := svc.UpdateCashFluxUser(context.Background(), &sitepb.CashFluxUpdateUserRequest{UserId: "device:owner", Role: "viewer"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v, want InvalidArgument", err)
	}
	if !strings.Contains(err.Error(), "cannot be demoted") {
		t.Fatalf("err = %v, want it to carry the reason", err)
	}
}

func TestSuspendAndResetPassThrough(t *testing.T) {
	fake := &fakeCashFluxAdmin{}
	svc := newCashFluxTestService(t, fake)
	if _, err := svc.SuspendCashFluxUser(context.Background(), &sitepb.CashFluxSuspendUserRequest{UserId: "u1", Suspended: true}); err != nil {
		t.Fatalf("SuspendCashFluxUser: %v", err)
	}
	if fake.suspendUserID != "u1" || !fake.suspendSuspended {
		t.Fatalf("suspend got id=%q suspended=%v", fake.suspendUserID, fake.suspendSuspended)
	}
	if _, err := svc.ResetCashFluxCredentials(context.Background(), &sitepb.CashFluxUserRef{UserId: "u2"}); err != nil {
		t.Fatalf("ResetCashFluxCredentials: %v", err)
	}
	if fake.resetUserID != "u2" {
		t.Fatalf("reset got id=%q", fake.resetUserID)
	}
}

func TestCashFluxCRUDUnconfigured(t *testing.T) {
	svc := newCashFluxTestService(t, nil)
	ctx := context.Background()
	if _, err := svc.CreateCashFluxUser(ctx, &sitepb.CashFluxCreateUserRequest{Username: "x"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Create: %v", err)
	}
	if _, err := svc.UpdateCashFluxUser(ctx, &sitepb.CashFluxUpdateUserRequest{UserId: "x"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Update: %v", err)
	}
	if _, err := svc.SuspendCashFluxUser(ctx, &sitepb.CashFluxSuspendUserRequest{UserId: "x"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Suspend: %v", err)
	}
	if _, err := svc.ResetCashFluxCredentials(ctx, &sitepb.CashFluxUserRef{UserId: "x"}); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Reset: %v", err)
	}
}

// TestMintCashFluxActivationCodeForUserTargetsTheAccount proves the console can hand an invited
// person a code for THEIR account. Without it, CreateUser produced an account nobody could sign
// in to, because the only other minting path always binds to the owner.
func TestMintCashFluxActivationCodeForUserTargetsTheAccount(t *testing.T) {
	expires := time.Date(2026, 7, 24, 23, 0, 0, 0, time.UTC)
	fake := &fakeCashFluxAdmin{forUserCode: "314159", forUserExpires: expires}
	svc := newCashFluxTestService(t, fake)
	resp, err := svc.MintCashFluxActivationCodeForUser(context.Background(), &sitepb.CashFluxUserRef{UserId: "device:PRIYA"})
	if err != nil {
		t.Fatalf("MintCashFluxActivationCodeForUser: %v", err)
	}
	if resp.GetCode() != "314159" || fake.forUserID != "device:PRIYA" {
		t.Fatalf("code=%q forUserID=%q", resp.GetCode(), fake.forUserID)
	}
	if got, want := resp.GetExpiresAt(), expires.Format(time.RFC3339); got != want {
		t.Fatalf("expiresAt = %q, want %q", got, want)
	}
}

func TestMintCashFluxActivationCodeForUnknownAccountIsInvalidArgument(t *testing.T) {
	fake := &fakeCashFluxAdmin{forUserErr: errors.New(`no such account "device:TYPO"`)}
	svc := newCashFluxTestService(t, fake)
	_, err := svc.MintCashFluxActivationCodeForUser(context.Background(), &sitepb.CashFluxUserRef{UserId: "device:TYPO"})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("err = %v, want InvalidArgument", err)
	}
}

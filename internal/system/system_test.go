package system

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// recorder captures what RecordCommand would have persisted, so the allowlist can be tested for
// what it *keeps out* rather than only for what it lets through.
type recorder struct{ names []string }

func (r *recorder) RecordCommand(_ context.Context, name string) error {
	r.names = append(r.names, name)
	return nil
}

func TestPingReturnsServerClock(t *testing.T) {
	svc := New(nil)
	before := time.Now().UnixMilli()
	resp, err := svc.Ping(context.Background(), &sitepb.PingRequest{})
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if resp.GetServerUnixMillis() < before {
		t.Fatalf("server clock %d predates the call at %d", resp.GetServerUnixMillis(), before)
	}
}

func TestGetStatsReportsLiveProcessFacts(t *testing.T) {
	svc := New(nil)
	svc.CountRequest()
	svc.CountRequest()
	st, err := svc.GetStats(context.Background(), &sitepb.StatsRequest{})
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if st.GetRequestsServed() != 2 {
		t.Errorf("requests = %d; want 2", st.GetRequestsServed())
	}
	if st.GetGoVersion() == "" {
		t.Error("go version should be reported")
	}
	if st.GetVersion() == "" {
		t.Error("build version should never be empty — it defaults to \"dev\"")
	}
	if st.GetGoroutines() < 1 {
		t.Errorf("goroutines = %d; want at least 1", st.GetGoroutines())
	}
	if st.GetStartedUnixMillis() == 0 {
		t.Error("start time should be set")
	}
}

func TestRecordCommandDropsAnythingNotAShippedCommand(t *testing.T) {
	// This is the privacy mechanism, so it is tested as one: the inputs below are the shapes that
	// a bug, a crafted client, or a careless future caller could produce, and none of them may be
	// stored. If this test ever has to be relaxed, the privacy claim in the proto has to change
	// with it.
	rec := &recorder{}
	svc := New(rec)
	for _, in := range []string{
		"projects",                // kept: a real command
		"open",                    // kept: a real command, without its argument
		"open cashflux",           // dropped: an argument rode along
		"my-password-is-hunter2",  // dropped: free text
		"cat notes/experience.md", // dropped: a whole line
		"",                        // dropped: empty
		"PROJECTS",                // dropped: not the shipped spelling
		"rm -rf ~",                // dropped: a line, not a name
	} {
		if _, err := svc.RecordCommand(context.Background(), &sitepb.CommandEvent{Name: in}); err != nil {
			t.Fatalf("RecordCommand(%q): %v", in, err)
		}
	}
	want := []string{"projects", "open"}
	if len(rec.names) != len(want) {
		t.Fatalf("stored %v; want exactly %v", rec.names, want)
	}
	for i, w := range want {
		if rec.names[i] != w {
			t.Errorf("stored[%d] = %q; want %q", i, rec.names[i], w)
		}
	}
}

func TestRecordCommandAlwaysSucceeds(t *testing.T) {
	// A visitor's command must never fail because a counter could not be written.
	svc := New(nil)
	ack, err := svc.RecordCommand(context.Background(), &sitepb.CommandEvent{Name: "not-a-command"})
	if err != nil || !ack.GetOk() {
		t.Fatalf("got ack=%v err=%v; want a successful ack", ack, err)
	}
}

func TestRecordCommandCountsOnlyKeptCommands(t *testing.T) {
	svc := New(nil)
	_, _ = svc.RecordCommand(context.Background(), &sitepb.CommandEvent{Name: "help"})
	_, _ = svc.RecordCommand(context.Background(), &sitepb.CommandEvent{Name: "junk"})
	st, _ := svc.GetStats(context.Background(), &sitepb.StatsRequest{})
	if st.GetTerminalCommands() != 1 {
		t.Fatalf("terminal commands = %d; want 1", st.GetTerminalCommands())
	}
}

func TestRecordCommandWriteBudget(t *testing.T) {
	// The RPC is public and unauthenticated, so a flood must not be able to hammer SQLite's single
	// writer. Past the per-second budget the in-memory counter keeps moving and the write is
	// dropped — the counter is what `stats` reports, and it is the cheaper thing to keep honest.
	rec := &recorder{}
	svc := New(rec)
	now := time.Now()
	for i := 0; i < maxWritesPerSecond*3; i++ {
		if _, err := svc.RecordCommand(context.Background(), &sitepb.CommandEvent{Name: "help"}); err != nil {
			t.Fatalf("RecordCommand: %v", err)
		}
	}
	if len(rec.names) > maxWritesPerSecond {
		t.Fatalf("wrote %d rows in one second; budget is %d", len(rec.names), maxWritesPerSecond)
	}
	st, _ := svc.GetStats(context.Background(), &sitepb.StatsRequest{})
	if st.GetTerminalCommands() != int64(maxWritesPerSecond*3) {
		t.Fatalf("counter = %d; every accepted command should still be counted", st.GetTerminalCommands())
	}
	// The window slides: a later second gets a fresh budget.
	if !svc.allowWrite(now.Add(2 * time.Second)) {
		t.Fatal("the budget should refill in the next second")
	}
}

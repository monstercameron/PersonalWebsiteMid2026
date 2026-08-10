// Package system serves live facts about the running process to the terminal's `stats`, `uptime`
// and boot-latency lines, and records the terminal's command counters.
//
// It exists because the terminal makes a claim — that this page is a real Go backend answering
// over a real gRPC tunnel — and a claim a visitor cannot check is just decoration. Everything here
// is measured from the process serving the request: uptime that ticks, a request counter that
// moves while you watch, the build the binary was made from.
package system

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// Version is the build identifier, set at link time by scripts/build.sh:
//
//	-ldflags "-X github.com/monstercameron/earlcameron/internal/system.Version=$(git rev-parse --short HEAD)"
//
// It stays "dev" for an unstamped local build, which is the honest answer for one.
var Version = "dev"

// commandRecorder is the persistence this service needs, kept as an interface so the service can
// be built and tested without a database.
type commandRecorder interface {
	RecordCommand(ctx context.Context, name string) error
}

// Service implements sitepb.SystemServiceServer.
type Service struct {
	sitepb.UnimplementedSystemServiceServer

	started  time.Time
	store    commandRecorder
	requests atomic.Int64
	commands atomic.Int64

	// mu guards the write budget below.
	mu          sync.Mutex
	windowStart time.Time
	windowCount int
}

// maxWritesPerSecond bounds how often RecordCommand touches the database.
//
// The RPC is public and unauthenticated, so anyone can call it in a loop. Row growth is already
// bounded — the primary key is (day, allowlisted name), so a flood updates existing rows rather
// than adding new ones — which means disk is not the risk. The write lock is: SQLite has a single
// writer, and a flood here would contend with the contact form's inserts on the same file. Past
// the budget the in-memory counter still moves (so `stats` stays truthful for the session) and the
// database write is skipped, which is the right thing to drop under load.
const maxWritesPerSecond = 20

// New builds the service. store may be nil, in which case command counts are still returned for
// the running process but nothing is persisted.
func New(store commandRecorder) *Service {
	return &Service{started: time.Now(), store: store}
}

// CountRequest records one served HTTP request. It is called from the server's middleware and is
// deliberately just a counter — no path, no client, nothing per-visitor.
func (s *Service) CountRequest() { s.requests.Add(1) }

// Ping answers the terminal's latency probe with the server's own clock, so the client can
// separate network time from server time instead of attributing the whole round trip to one.
func (s *Service) Ping(context.Context, *sitepb.PingRequest) (*sitepb.PingResponse, error) {
	return &sitepb.PingResponse{ServerUnixMillis: time.Now().UnixMilli()}, nil
}

// GetStats returns the live process facts behind `stats`.
func (s *Service) GetStats(context.Context, *sitepb.StatsRequest) (*sitepb.Stats, error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return &sitepb.Stats{
		Version:           Version,
		GoVersion:         runtime.Version(),
		UptimeSeconds:     int64(time.Since(s.started).Seconds()),
		StartedUnixMillis: s.started.UnixMilli(),
		RequestsServed:    s.requests.Load(),
		TerminalCommands:  s.commands.Load(),
		Goroutines:        int32(runtime.NumGoroutine()),
		HeapBytes:         int64(mem.HeapAlloc),
	}, nil
}

// RecordCommand counts one terminal command.
//
// The allowlist is the privacy mechanism, not a validation nicety: a name that is not a command
// this terminal ships is dropped rather than stored, so no amount of client-side creativity — a
// crafted RPC, a bug, a future caller that forgets — can turn this counter into a log of what
// visitors typed. It always reports success, because whether a counter was incremented is not a
// visitor's problem and a failure here must never surface as a broken command.
func (s *Service) RecordCommand(ctx context.Context, in *sitepb.CommandEvent) (*sitepb.Ack, error) {
	name := in.GetName()
	if !KnownCommands[name] {
		return &sitepb.Ack{Ok: true}, nil
	}
	s.commands.Add(1)
	if s.store != nil && s.allowWrite(time.Now()) {
		// Errors are swallowed on purpose: a full disk should not make the terminal look broken.
		_ = s.store.RecordCommand(ctx, name)
	}
	return &sitepb.Ack{Ok: true}, nil
}

// allowWrite reports whether another database write fits in this second's budget, recording it
// when it does. See maxWritesPerSecond for why the budget exists and why dropping is the right
// failure mode.
func (s *Service) allowWrite(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if now.Sub(s.windowStart) >= time.Second {
		s.windowStart, s.windowCount = now, 0
	}
	if s.windowCount >= maxWritesPerSecond {
		return false
	}
	s.windowCount++
	return true
}

// KnownCommands is the allowlist of command names that may be counted. It mirrors the terminal's
// vocabulary in client/shellutil.go (allCommands) plus the hidden commands, which are counted
// precisely because how often anyone finds them is the interesting question.
//
// Anything absent is discarded. Keep additions here to literal command names — never arguments.
var KnownCommands = map[string]bool{
	// portfolio programs
	"help": true, "about": true, "whoami": true, "projects": true, "open": true,
	"neofetch": true, "links": true, "resume": true, "anime": true, "contact": true,
	// terminal features
	"tour": true, "share": true, "theme": true, "stats": true, "uptime": true,
	"bench": true, "clear": true, "reset": true,
	// shell
	"pwd": true, "ls": true, "cd": true, "mkdir": true, "touch": true, "cp": true,
	"mv": true, "rm": true, "cat": true, "less": true, "head": true, "tail": true,
	"nano": true, "echo": true, "grep": true, "find": true, "sort": true, "uniq": true,
	"cut": true, "sed": true, "awk": true, "wc": true, "du": true, "df": true,
	"history": true, "curl": true, "ssh": true, "tar": true, "man": true,
	// second-tier coreutils
	"uname": true, "arch": true, "hostname": true, "id": true, "groups": true, "users": true,
	"who": true, "w": true, "date": true, "cal": true, "free": true, "lscpu": true, "ps": true,
	"env": true, "printenv": true, "which": true, "whereis": true, "type": true, "tree": true,
	"seq": true, "rev": true, "tac": true, "nl": true, "tr": true, "base64": true,
	"sha256sum": true, "md5sum": true, "xxd": true, "hexdump": true, "basename": true,
	"dirname": true, "realpath": true, "stat": true, "file": true, "diff": true, "factor": true,
	"expr": true, "shuf": true, "yes": true, "banner": true, "figlet": true, "alias": true,
	"sleep": true, "wget": true, "chmod": true, "chown": true, "exit": true, "logout": true,
	"quit": true, "ping": true,
	// hidden
	"sudo": true, "vim": true, "vi": true, "emacs": true, "cowsay": true, "fortune": true,
	"htop": true, "sl": true, "matrix": true, "cmatrix": true, "snake": true,
}

//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"syscall/js"
	"time"

	"github.com/monstercameron/earlcameron/proto/sitepb"
)

// The boot log used to print "wasm · 4.2 mb" and "tunnel established · 14 ms" as string literals.
// They were plausible, they were on the screen of a site whose entire pitch is that it really is
// Go over a real gRPC tunnel, and they were made up. Anyone who opened DevTools found that out.
//
// Everything below is measured from the browser and the server that answered: transfer size from
// the Resource Timing entry for the wasm binary, hydration from the navigation clock, latency from
// an actual round trip. The numbers are less flattering than the invented ones and worth far more.

// Boot-line indices in bootScrollback, so measured values can replace the line they belong to.
const (
	bootLineRuntime = 0
	bootLineSocket  = 1
)

// measureBoot fills in the boot log's real numbers: the wasm transfer size and hydration time
// immediately, and the tunnel round-trip once the probe returns.
func measureBoot(t *termCtl) {
	t.replaceAt(bootLineRuntime, bootLine("loading runtime", wasmSizeText()+" · hydrated in "+hydrationText()))

	go func() {
		rtt, serverSkew, err := pingTunnel()
		if err != nil {
			// An offline or blocked tunnel is a real state the boot log should show, not hide
			// behind a comfortable number.
			t.replaceAt(bootLineSocket, bootLine("dialing /socket", "unreachable — offline?"))
			return
		}
		detail := fmt.Sprintf("gRPC-over-ws · %d ms round trip", rtt.Milliseconds())
		if serverSkew > 0 {
			detail += fmt.Sprintf(" (%d ms to the server)", serverSkew.Milliseconds())
		}
		t.replaceAt(bootLineSocket, bootLine("dialing /socket", detail))
	}()
}

// pingTunnel measures a real gRPC round trip over the WebSocket tunnel. It returns the total round
// trip and, when the server's clock is close enough to the browser's for the comparison to mean
// anything, the outbound leg.
func pingTunnel() (time.Duration, time.Duration, error) {
	cl, err := systemClient()
	if err != nil {
		return 0, 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	start := time.Now()
	resp, err := cl.Ping(ctx, &sitepb.PingRequest{})
	if err != nil {
		return 0, 0, err
	}
	rtt := time.Since(start)
	// The server's clock is not synchronised with the browser's, so the outbound leg is only
	// reported when it lands inside the round trip — otherwise the two clocks disagree by more
	// than the measurement is worth and printing it would be inventing precision.
	outbound := time.UnixMilli(resp.GetServerUnixMillis()).Sub(start)
	if outbound <= 0 || outbound > rtt {
		return rtt, 0, nil
	}
	return rtt, outbound, nil
}

// wasmSizeText reports how many bytes the wasm binary actually cost to fetch, from the browser's
// own Resource Timing entry. A cached or compressed response reports what it really was.
func wasmSizeText() string {
	perf := js.Global().Get("performance")
	if !perf.Truthy() || !perf.Get("getEntriesByType").Truthy() {
		return "wasm"
	}
	entries := perf.Call("getEntriesByType", "resource")
	for i := 0; i < entries.Get("length").Int(); i++ {
		e := entries.Index(i)
		name := e.Get("name").String()
		if len(name) < 5 || name[len(name)-5:] != ".wasm" {
			continue
		}
		transfer := e.Get("transferSize")
		encoded := e.Get("encodedBodySize")
		decoded := e.Get("decodedBodySize")
		switch {
		case transfer.Type() == js.TypeNumber && transfer.Int() > 0:
			return "wasm " + humanBytes(transfer.Int()) + " over the wire"
		case encoded.Type() == js.TypeNumber && encoded.Int() > 0:
			return "wasm " + humanBytes(encoded.Int()) + " (cached)"
		case decoded.Type() == js.TypeNumber && decoded.Int() > 0:
			return "wasm " + humanBytes(decoded.Int())
		}
	}
	return "wasm"
}

// hydrationText reports how long it took from navigation to this code running.
func hydrationText() string {
	perf := js.Global().Get("performance")
	if !perf.Truthy() || !perf.Get("now").Truthy() {
		return "?"
	}
	return fmt.Sprintf("%d ms", int(perf.Call("now").Float()))
}

// humanBytes formats a byte count for the boot log.
func humanBytes(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

// goVersionString returns the Go toolchain this binary was built with.
func goVersionString() string { return runtime.Version() }

// startStats implements `stats` and `uptime`: live facts from the process serving the page.
func startStats(name string, t *termCtl) {
	tok := t.token()
	t.emitRows([]progRow{dim("asking the server…")})
	go func() {
		// The failure paths check the token too. They are the slow ones — an unreachable tunnel
		// fails on a timeout, by which point the visitor has almost certainly typed something
		// else, and writing then would clobber that command's output with a stale error.
		fail := func(msg string) {
			if !t.live(tok) {
				return
			}
			t.replaceLast(renderRow(progRow{text: msg, style: rowErr}))
			t.emit(gap())
		}
		cl, err := systemClient()
		if err != nil {
			fail("stats: tunnel unavailable — " + err.Error())
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		start := time.Now()
		st, err := cl.GetStats(ctx, &sitepb.StatsRequest{})
		if err != nil {
			fail("stats: " + err.Error())
			return
		}
		if !t.live(tok) {
			return
		}
		rtt := time.Since(start)
		if name == "uptime" {
			t.replaceLast(renderRow(row(uptimeLine(st))))
			t.emit(gap())
			return
		}
		t.replaceLast(renderRow(head("live from the server")))
		t.emitRows(statsRows(st, rtt))
		t.emit(gap())
	}()
}

// uptimeLine renders the one-line `uptime` answer, in the shape the real command uses.
func uptimeLine(st *sitepb.Stats) string {
	return fmt.Sprintf("up %s · build %s · %d requests served",
		humanDuration(time.Duration(st.GetUptimeSeconds())*time.Second), st.GetVersion(), st.GetRequestsServed())
}

// statsRows renders the full `stats` table.
func statsRows(st *sitepb.Stats, rtt time.Duration) []progRow {
	return []progRow{
		kv("build", 150, st.GetVersion()+"  ·  "+st.GetGoVersion()),
		kv("uptime", 150, humanDuration(time.Duration(st.GetUptimeSeconds())*time.Second)+
			"  (since "+time.UnixMilli(st.GetStartedUnixMillis()).Format("2006-01-02 15:04 MST")+")"),
		kv("requests", 150, fmt.Sprintf("%d served", st.GetRequestsServed())),
		kv("commands", 150, fmt.Sprintf("%d run in this terminal since boot", st.GetTerminalCommands())),
		kv("goroutines", 150, fmt.Sprintf("%d", st.GetGoroutines())),
		kv("heap", 150, humanBytes(int(st.GetHeapBytes()))),
		kv("this call", 150, fmt.Sprintf("%d ms round trip over gRPC-over-WebSocket", rtt.Milliseconds())),
		dim("run it again — the uptime and request count move."),
	}
}

// humanDuration formats a duration the way `uptime` does.
func humanDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	case mins > 0:
		return fmt.Sprintf("%dm %ds", mins, secs)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}

// pingCount is how many probes `ping` sends. Four, like the Windows default — enough to see
// variance, few enough to finish while someone is still watching.
const pingCount = 4

// startPing sends real round trips over the gRPC tunnel and prints each one, then a summary. It is
// `ping` doing what ping does: there is no ICMP in a browser, but there is a real server at the
// other end of a real socket, and timing it is the honest equivalent.
func startPing(t *termCtl) {
	tok := t.token()
	t.emitRows([]progRow{dim("PING /socket (gRPC-over-WebSocket) — " + strconv.Itoa(pingCount) + " probes")})
	go func() {
		var times []time.Duration
		for i := 0; i < pingCount; i++ {
			if !t.live(tok) {
				return
			}
			rtt, _, err := pingTunnel()
			if err != nil {
				if t.live(tok) {
					t.emitRows([]progRow{{text: "ping: " + err.Error(), style: rowErr}})
					t.emit(gap())
				}
				return
			}
			times = append(times, rtt)
			t.emitRows([]progRow{row(fmt.Sprintf("reply from /socket: seq=%d time=%d ms", i+1, rtt.Milliseconds()))})
			// A pause between probes, so this reads as a sequence of measurements rather than one
			// burst — and so it never monopolises the tunnel.
			done := make(chan struct{})
			after(320, func() { close(done) })
			<-done
		}
		if !t.live(tok) || len(times) == 0 {
			return
		}
		min, max, total := times[0], times[0], time.Duration(0)
		for _, d := range times {
			if d < min {
				min = d
			}
			if d > max {
				max = d
			}
			total += d
		}
		t.emitRows([]progRow{
			dim(fmt.Sprintf("--- %d probes, 0%% loss ---", len(times))),
			row(fmt.Sprintf("rtt min/avg/max = %d/%d/%d ms",
				min.Milliseconds(), (total / time.Duration(len(times))).Milliseconds(), max.Milliseconds())),
		})
		t.emit(gap())
	}()
}

// openResume opens the résumé document. The project serves a print-optimised HTML page rather than
// a static PDF (TODOS §8) — "save as PDF" is the browser's job — so this navigates to it in a new
// tab, leaving whatever the visitor built up in the terminal intact behind them.
func openResume() { openInNewTab("/resume") }

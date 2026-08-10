//go:build js && wasm

package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"syscall/js"
	"time"
)

// `bench` measures this wasm runtime, in this browser, on this machine, right now. It is the one
// command on the site a static page cannot fake: the numbers change with the visitor's hardware,
// and anyone who doubts them can run it twice.
//
// Each case runs for a bounded slice of wall-clock time and then yields to the event loop before
// the next one starts. A tight loop that runs to completion in one go would freeze the tab —
// wasm shares the browser's single thread, so "fast" here means "gives the frame back".

// benchBudgetMs is how long each individual case is allowed to run.
const benchBudgetMs = 120

// benchCase is one measurement: a name, a unit of work, and the label for what one op is.
type benchCase struct {
	name string
	unit string
	// fn performs n units of work. Returning a value keeps the compiler from optimising the body
	// away, which is the classic way a microbenchmark ends up measuring nothing at all.
	fn func(n int) int
}

// benchCases are chosen to say something about the stack rather than to produce a big number:
// raw compute, allocation, the encoding path every gRPC message pays, and the wasm↔JS boundary
// that any Go-in-the-browser framework lives or dies on.
var benchCases = []benchCase{
	{"integer math", "ops", benchInt},
	{"slice alloc", "allocs", benchAlloc},
	{"json round-trip", "docs", benchJSON},
	{"wasm↔js call", "calls", benchJSCall},
	{"string build", "concats", benchStrings},
}

// startBench runs the suite, printing each result as it lands.
func startBench(t *termCtl) {
	tok := t.token()
	t.emitRows([]progRow{
		head("bench — measuring this runtime, live"),
		dim("Go " + goVersionString() + " · wasm · " + hardwareBlurb()),
	})
	runBenchCase(t, tok, 0)
}

// runBenchCase runs one case, prints it, and schedules the next. The yield between cases is what
// keeps the page responsive while the suite runs.
func runBenchCase(t *termCtl, tok, i int) {
	if !t.live(tok) {
		return
	}
	if i >= len(benchCases) {
		t.emitRows([]progRow{dim("measured in your browser just now — run it again and watch it move.")})
		t.emit(gap())
		return
	}
	c := benchCases[i]
	ops, elapsed := measure(c.fn)
	perSec := float64(ops) / elapsed.Seconds()
	t.emitRows([]progRow{kv(c.name, 150, fmt.Sprintf("%s %s/sec", humanRate(perSec), c.unit))})
	after(16, func() { runBenchCase(t, tok, i+1) })
}

// measure runs fn in growing batches until the time budget is spent, and reports the total work
// done and the time it took. Batching amortises the clock read, which on a wasm runtime is not
// cheap enough to call per operation.
func measure(fn func(n int) int) (int, time.Duration) {
	batch := 1024
	total := 0
	start := time.Now()
	deadline := start.Add(benchBudgetMs * time.Millisecond)
	sink := 0
	for time.Now().Before(deadline) {
		sink += fn(batch)
		total += batch
		if batch < 1<<20 {
			batch *= 2
		}
	}
	benchSink = sink
	return total, time.Since(start)
}

// benchSink holds each case's result so the work cannot be eliminated as dead code.
var benchSink int

// benchInt is straight integer arithmetic — the compiler's baseline.
func benchInt(n int) int {
	x := 0
	for i := 0; i < n; i++ {
		x = x*31 + i ^ (x >> 3)
	}
	return x
}

// benchAlloc measures allocation and GC pressure.
func benchAlloc(n int) int {
	total := 0
	for i := 0; i < n; i++ {
		b := make([]byte, 64)
		b[i%64] = byte(i)
		total += len(b)
	}
	return total
}

// benchDoc is the struct benchJSON round-trips — roughly the shape of a small gRPC message.
type benchDoc struct {
	ID    int      `json:"id"`
	Name  string   `json:"name"`
	Tags  []string `json:"tags"`
	Score float64  `json:"score"`
}

// benchJSON measures the encode/decode path.
func benchJSON(n int) int {
	doc := benchDoc{ID: 7, Name: "cashflux", Tags: []string{"go", "wasm", "grpc"}, Score: 0.94}
	total := 0
	for i := 0; i < n; i++ {
		b, err := json.Marshal(doc)
		if err != nil {
			return total
		}
		var out benchDoc
		if err := json.Unmarshal(b, &out); err != nil {
			return total
		}
		total += out.ID
	}
	return total
}

// benchJSCall measures the wasm↔JS boundary, which is the tax every DOM write pays.
func benchJSCall(n int) int {
	math := js.Global().Get("Math")
	total := 0
	for i := 0; i < n; i++ {
		total += math.Call("abs", -i).Int()
	}
	return total
}

// benchStrings measures buffered string building.
func benchStrings(n int) int {
	var b strings.Builder
	for i := 0; i < n; i++ {
		b.WriteString("x")
	}
	return b.Len()
}

// humanRate formats a per-second rate compactly.
func humanRate(v float64) string {
	switch {
	case v >= 1e9:
		return fmt.Sprintf("%.2f B", v/1e9)
	case v >= 1e6:
		return fmt.Sprintf("%.2f M", v/1e6)
	case v >= 1e3:
		return fmt.Sprintf("%.1f K", v/1e3)
	default:
		return fmt.Sprintf("%.0f", v)
	}
}

// hardwareBlurb reports what the browser is willing to say about the machine. Both values are
// deliberately coarse in every browser, so this is a hint, not a fingerprint.
func hardwareBlurb() string {
	nav := js.Global().Get("navigator")
	if !nav.Truthy() {
		return "unknown hardware"
	}
	cores := nav.Get("hardwareConcurrency")
	if cores.Type() != js.TypeNumber {
		return "unknown hardware"
	}
	return fmt.Sprintf("%d logical cores", cores.Int())
}

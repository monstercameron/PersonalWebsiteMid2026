//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"testing"
	"time"
)

// The client package is GOOS=js/GOARCH=wasm only, so these run under node via
// scripts/test-wasm.sh (and the same step in CI) rather than under a plain `go test ./...`.
// Everything tested here is pure logic — the parser, the history, the line editor, the virtual
// filesystem — which is exactly the layer AGENTS.md asks to be unit-tested without a browser.

// TestMain installs the browser globals the code under test reaches for. node has no
// localStorage, and the virtual filesystem reads it on construction.
func TestMain(m *testing.M) {
	installLocalStorage()
	m.Run()
}

// installLocalStorage provides an in-memory localStorage shim.
func installLocalStorage() {
	store := map[string]string{}
	obj := js.Global().Get("Object").New()
	obj.Set("getItem", js.FuncOf(func(_ js.Value, a []js.Value) any {
		if v, ok := store[a[0].String()]; ok {
			return v
		}
		return js.Null()
	}))
	obj.Set("setItem", js.FuncOf(func(_ js.Value, a []js.Value) any {
		store[a[0].String()] = a[1].String()
		return nil
	}))
	obj.Set("removeItem", js.FuncOf(func(_ js.Value, a []js.Value) any {
		delete(store, a[0].String())
		return nil
	}))
	js.Global().Set("localStorage", obj)
}

// --- history ---

func TestHistNavWalksAndRestoresDraft(t *testing.T) {
	var h histNav
	h.record("ls")
	h.record("projects")

	got, ok := h.up("half-typed")
	if !ok || got != "projects" {
		t.Fatalf("first up = %q, %v; want projects", got, ok)
	}
	got, ok = h.up(got)
	if !ok || got != "ls" {
		t.Fatalf("second up = %q, %v; want ls", got, ok)
	}
	if _, ok := h.up(got); ok {
		t.Fatal("up past the oldest entry should report no move")
	}
	got, _ = h.down()
	if got != "projects" {
		t.Fatalf("down = %q; want projects", got)
	}
	// Past the newest entry, the half-typed line the visitor abandoned comes back.
	got, ok = h.down()
	if !ok || got != "half-typed" {
		t.Fatalf("down to draft = %q, %v; want half-typed", got, ok)
	}
	if _, ok := h.down(); ok {
		t.Fatal("down past the draft should report no move")
	}
}

func TestHistNavCollapsesConsecutiveDuplicates(t *testing.T) {
	var h histNav
	h.record("ls")
	h.record("ls")
	h.record("ls")
	if len(h.entries) != 1 {
		t.Fatalf("entries = %v; want one collapsed entry", h.entries)
	}
}

func TestHistNavEmptyHistory(t *testing.T) {
	var h histNav
	if _, ok := h.up("x"); ok {
		t.Fatal("up on an empty history should report no move")
	}
	if _, ok := h.down(); ok {
		t.Fatal("down on an empty history should report no move")
	}
}

func TestHistNavRecordResetsBrowsing(t *testing.T) {
	var h histNav
	h.record("a")
	h.record("b")
	if _, ok := h.up(""); !ok {
		t.Fatal("expected to move up")
	}
	h.record("c")
	// After running a command the cursor is back at the end, so one Up gives the newest entry.
	got, ok := h.up("")
	if !ok || got != "c" {
		t.Fatalf("up after record = %q, %v; want c", got, ok)
	}
}

// --- line editing ---

func TestKillToLineStart(t *testing.T) {
	line, caret := killToLineStart("cat notes/about.md", 4)
	if line != "notes/about.md" || caret != 0 {
		t.Fatalf("got %q,%d; want %q,0", line, caret, "notes/about.md")
	}
}

func TestKillPrevWord(t *testing.T) {
	for _, tc := range []struct {
		name      string
		line      string
		caret     int
		want      string
		wantCaret int
	}{
		{"mid line", "open cashflux", 13, "open ", 5},
		{"trailing space", "open cashflux ", 14, "open ", 5},
		{"first word", "open", 4, "", 0},
		{"caret inside", "open cashflux", 5, "cashflux", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, caret := killPrevWord(tc.line, tc.caret)
			if got != tc.want || caret != tc.wantCaret {
				t.Fatalf("got %q,%d; want %q,%d", got, caret, tc.want, tc.wantCaret)
			}
		})
	}
}

func TestClampCaretHandlesMissingSelection(t *testing.T) {
	// -1 is what caretPos reports when the DOM gives no selectionStart; it must mean "end of
	// line", not a panic or a truncated edit.
	if got := clampCaret("abc", -1); got != 3 {
		t.Fatalf("clampCaret(-1) = %d; want 3", got)
	}
	if got := clampCaret("abc", 99); got != 3 {
		t.Fatalf("clampCaret(99) = %d; want 3", got)
	}
}

// --- did you mean ---

func TestDidYouMean(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"lss", "ls"},
		{"projets", "projects"},
		{"hepl", "help"},
		{"neofetc", "neofetch"},
		{"xyzzy", ""},
		{"", ""},
	} {
		if got := didYouMean(tc.in); got != tc.want {
			t.Errorf("didYouMean(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	for _, tc := range []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"ls", "ls", 0},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
	} {
		if got := levenshtein(tc.a, tc.b); got != tc.want {
			t.Errorf("levenshtein(%q,%q) = %d; want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

// --- virtual filesystem ---

func TestRepairSeedRestoresDeletedNotes(t *testing.T) {
	v := &vfs{root: seedTree(), cwd: []string{"home", "cam"}}
	// The defect this guards: a visitor runs `rm -rf ~/notes`, and every later visit finds the
	// recruiter-facing content gone for good.
	if err := v.rm("notes", true); err != nil {
		t.Fatalf("rm: %v", err)
	}
	if v.get(v.resolve("notes")) != nil {
		t.Fatal("notes should be gone within the session")
	}
	if !repairSeed(v.root, seedTree()) {
		t.Fatal("repairSeed should report that it restored something")
	}
	n := v.get(v.resolve("notes/experience.md"))
	if n == nil || n.Content == "" {
		t.Fatal("notes/experience.md should have been restored")
	}
}

func TestRepairSeedKeepsVisitorEdits(t *testing.T) {
	v := &vfs{root: seedTree(), cwd: []string{"home", "cam"}}
	if err := v.writeFile("notes/README.md", "mine now"); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	if err := v.mkdir("scratch"); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	repairSeed(v.root, seedTree())
	if got := v.get(v.resolve("notes/README.md")).Content; got != "mine now" {
		t.Fatalf("edit was overwritten: %q", got)
	}
	if v.get(v.resolve("scratch")) == nil {
		t.Fatal("a directory the visitor created should survive repair")
	}
}

func TestResetRestoresEverything(t *testing.T) {
	v := &vfs{root: seedTree(), cwd: []string{"home", "cam"}}
	_ = v.rm("notes", true)
	_ = v.mkdir("junk")
	v.reset()
	if v.get(v.resolve("notes/skills.md")) == nil {
		t.Fatal("reset should restore the seeded tree")
	}
	if v.get(v.resolve("junk")) != nil {
		t.Fatal("reset should discard what the visitor made")
	}
	if v.pwd() != "/home/cam" {
		t.Fatalf("reset should return home; pwd = %q", v.pwd())
	}
}

// --- shell ---

func TestPortfolioProgramsPipe(t *testing.T) {
	// The defect: `projects | grep Go` used to produce nothing, because a portfolio program was
	// only ever rendered as styled nodes and never as text a pipeline could consume.
	sh := newShell()
	out := sh.run("projects | grep GoWebComponents")
	if len(out) != 1 || out[0].isErr {
		t.Fatalf("unexpected output: %+v", out)
	}
	if !strings.Contains(out[0].text, "GoWebComponents") {
		t.Fatalf("piped output missing the match: %q", out[0].text)
	}
	if strings.Contains(out[0].text, "CashFlux") {
		t.Fatalf("grep did not filter: %q", out[0].text)
	}
}

func TestPortfolioProgramRedirectsToFile(t *testing.T) {
	sh := newShell()
	if out := sh.run("about > notes/cam.md"); len(out) != 0 {
		t.Fatalf("redirect should print nothing, got %+v", out)
	}
	n := sh.fs.get(sh.fs.resolve("notes/cam.md"))
	if n == nil || !strings.Contains(n.Content, "AI-first engineer") {
		t.Fatalf("redirected content missing: %+v", n)
	}
}

func TestUnknownCommandSuggests(t *testing.T) {
	sh := newShell()
	out := sh.run("projets")
	if len(out) != 1 || !out[0].isErr {
		t.Fatalf("expected one error line, got %+v", out)
	}
	if !strings.Contains(out[0].text, "did you mean `projects`") {
		t.Fatalf("missing suggestion: %q", out[0].text)
	}
}

func TestResetCommandThroughShell(t *testing.T) {
	sh := newShell()
	if out := sh.run("rm -r notes"); len(out) != 0 {
		t.Fatalf("rm printed %+v", out)
	}
	out := sh.run("reset")
	if len(out) != 1 || out[0].isErr {
		t.Fatalf("reset failed: %+v", out)
	}
	if sh.fs.get(sh.fs.resolve("notes/skills.md")) == nil {
		t.Fatal("reset should have restored notes/")
	}
}

func TestHistoryRecordsPortfolioPrograms(t *testing.T) {
	// History used to be recorded inside shell.run, which portfolio programs never reach — so
	// `help` then `history` showed nothing at all.
	sh := newShell()
	sh.nav.record("help")
	sh.nav.record("ls")
	out := sh.run("history")
	if len(out) != 1 || !strings.Contains(out[0].text, "help") {
		t.Fatalf("history missing a portfolio program: %+v", out)
	}
}

// --- completion ---

func TestCompleteCommandAndPath(t *testing.T) {
	sh := newShell()
	if got := complete("neofet", sh); got != "neofetch " {
		t.Fatalf("command completion = %q", got)
	}
	if got := complete("cat notes/exp", sh); got != "cat notes/experience.md " {
		t.Fatalf("path completion = %q", got)
	}
}

// --- deep links ---

func TestDeepLinkAllowsOnlyReadOnlyCommands(t *testing.T) {
	for _, tc := range []struct {
		cmd  string
		want bool
	}{
		{"projects", true},
		{"open cashflux", true},
		{"cat notes/experience.md", true},
		{"rm -rf ~", false},          // the reason the allowlist exists
		{"echo hi > notes/x", false}, // a redirect writes, so the operator check must catch it
		{"projects | grep Go", false},
		{"reset", false},
		{"contact", false}, // interactive, and sends: never from a stranger's link
		{"", false},
	} {
		if got := deepLinkAllowed(tc.cmd); got != tc.want {
			t.Errorf("deepLinkAllowed(%q) = %v; want %v", tc.cmd, got, tc.want)
		}
	}
}

// --- program rendering ---

func TestRowsTextRendersColumnsAndLinks(t *testing.T) {
	text := rowsText(openOut([]string{"cashflux"}))
	for _, want := range []string{"CashFlux", "github.com/monstercameron/CashFlux", "/projects/cashflux"} {
		if !strings.Contains(text, want) {
			t.Errorf("open cashflux text missing %q:\n%s", want, text)
		}
	}
}

func TestOpenUnknownProjectSuggests(t *testing.T) {
	text := rowsText(openOut([]string{"cashflx"}))
	if !strings.Contains(text, "open `cashflux`") && !strings.Contains(text, "open cashflux") {
		t.Fatalf("expected a suggestion, got:\n%s", text)
	}
}

func TestOpenOnlyLinksExistingCaseStudies(t *testing.T) {
	// A link to /projects/<slug> for a project with no page is exactly the dead-end link the
	// recruiter pass (TODOS §13.B) is about.
	text := rowsText(openOut([]string{"semanticscript"}))
	if strings.Contains(text, "/projects/semanticscript") {
		t.Fatalf("linked a case study that does not exist:\n%s", text)
	}
}

func TestHelpListsNoEasterEggs(t *testing.T) {
	text := rowsText(helpOut())
	for _, egg := range []string{"cowsay", "sudo", "matrix", "snake", "fortune"} {
		if strings.Contains(text, egg) {
			t.Errorf("help should not list the hidden command %q", egg)
		}
	}
}

// --- formatting helpers ---

func TestWrapTextAndCowsay(t *testing.T) {
	got := cowsay("hello there this is a much longer line than the bubble is wide, by design")
	if !strings.Contains(got, "^__^") {
		t.Fatalf("missing the cow:\n%s", got)
	}
	for _, line := range strings.Split(got, "\n") {
		if len(line) > 60 {
			t.Fatalf("line ran away: %q", line)
		}
	}
}

func TestHumanBytes(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want string
	}{
		{512, "512 B"},
		{2048, "2 KB"},
		{3 * 1024 * 1024, "3.0 MB"},
	} {
		if got := humanBytes(tc.in); got != tc.want {
			t.Errorf("humanBytes(%d) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// --- cp/mv destructiveness (defects found in adversarial review) ---

func TestCopyOntoDirectoryGoesInsideIt(t *testing.T) {
	// The defect: `cp notes/README.md notes` replaced the entire notes/ directory with a single
	// file, destroying every recruiter-facing document in one command, silently.
	sh := newShell()
	if out := sh.run("cp about.md notes"); len(out) != 0 && out[0].isErr {
		t.Fatalf("cp failed: %+v", out)
	}
	if n := sh.fs.get(sh.fs.resolve("notes")); n == nil || !n.Dir {
		t.Fatal("notes/ must still be a directory")
	}
	if sh.fs.get(sh.fs.resolve("notes/experience.md")) == nil {
		t.Fatal("cp onto a directory destroyed its contents")
	}
	if sh.fs.get(sh.fs.resolve("notes/about.md")) == nil {
		t.Fatal("the copy should have landed inside notes/")
	}
}

// TestCopyFileOntoItsOwnDirectoryIsRefused matches real cp, which refuses rather than clobbering.
func TestCopyFileOntoItsOwnDirectoryIsRefused(t *testing.T) {
	sh := newShell()
	out := sh.run("cp notes/README.md notes")
	if len(out) != 1 || !out[0].isErr || !strings.Contains(out[0].text, "same file") {
		t.Fatalf("expected a same-file refusal, got %+v", out)
	}
	if sh.fs.get(sh.fs.resolve("notes/experience.md")) == nil {
		t.Fatal("the refusal must not have damaged notes/")
	}
}

func TestMoveOntoItselfIsANoOp(t *testing.T) {
	// The defect: mv was copy-then-remove with no same-path guard, so `mv x x` deleted x.
	sh := newShell()
	before := sh.fs.get(sh.fs.resolve("notes/README.md")).Content
	if out := sh.run("mv notes/README.md notes/README.md"); len(out) != 0 && out[0].isErr {
		t.Fatalf("mv failed: %+v", out)
	}
	n := sh.fs.get(sh.fs.resolve("notes/README.md"))
	if n == nil {
		t.Fatal("mv onto itself deleted the file")
	}
	if n.Content != before {
		t.Fatal("mv onto itself changed the file")
	}
}

func TestMoveIntoDirectoryKeepsTheDirectory(t *testing.T) {
	sh := newShell()
	if out := sh.run("mv about.md notes"); len(out) != 0 && out[0].isErr {
		t.Fatalf("mv failed: %+v", out)
	}
	if sh.fs.get(sh.fs.resolve("notes/about.md")) == nil {
		t.Fatal("the file should have landed inside notes/")
	}
	if sh.fs.get(sh.fs.resolve("notes/skills.md")) == nil {
		t.Fatal("mv onto a directory destroyed its contents")
	}
	if sh.fs.get(sh.fs.resolve("about.md")) != nil {
		t.Fatal("the source should be gone after mv")
	}
}

// --- second-tier coreutils ---

func TestUnameExistsAndIsHonest(t *testing.T) {
	// `uname` not existing is what started this: a visitor typed it and the did-you-mean
	// suggested `anime`.
	sh := newShell()
	out := sh.run("uname -a")
	if len(out) != 1 || out[0].isErr {
		t.Fatalf("uname failed: %+v", out)
	}
	if !strings.Contains(out[0].text, "wasm") {
		t.Fatalf("uname should say what this really is: %q", out[0].text)
	}
	if didYouMean("uname") != "uname" {
		t.Fatalf("uname should resolve to itself, got %q", didYouMean("uname"))
	}
}

func TestCoreutilsProduceOutput(t *testing.T) {
	sh := newShell()
	for _, cmd := range []string{
		"date", "cal", "free", "lscpu", "ps", "env", "id", "hostname", "groups", "who",
		"arch", "tree", "alias", "banner hi", "factor 12", "expr 6 * 7", "seq 3",
		"which ls", "basename a/b/c.md", "dirname a/b/c.md", "stat about.md", "file about.md",
	} {
		out := sh.run(cmd)
		if len(out) != 1 || out[0].isErr || strings.TrimSpace(out[0].text) == "" {
			t.Errorf("%q produced nothing useful: %+v", cmd, out)
		}
	}
}

func TestCoreutilsThatTransformText(t *testing.T) {
	sh := newShell()
	for _, tc := range []struct{ cmd, want string }{
		{"echo hello | rev", "olleh"},
		{"echo abc | base64", "YWJjCg=="},
		{"echo 12 | tr 12 34", "34"},
		{"expr 6 * 7", "42"},
		{"factor 12", "12: 2 2 3"},
		{"seq 3", "1\n2\n3"},
	} {
		out := sh.run(tc.cmd)
		if len(out) != 1 || out[0].isErr {
			t.Errorf("%q errored: %+v", tc.cmd, out)
			continue
		}
		if !strings.Contains(out[0].text, tc.want) {
			t.Errorf("%q = %q; want it to contain %q", tc.cmd, out[0].text, tc.want)
		}
	}
}

func TestBase64RoundTrips(t *testing.T) {
	sh := newShell()
	out := sh.run("echo terminal | base64 | base64 -d")
	if len(out) != 1 || out[0].isErr {
		t.Fatalf("round trip errored: %+v", out)
	}
	if !strings.Contains(out[0].text, "terminal") {
		t.Fatalf("round trip lost the input: %q", out[0].text)
	}
}

func TestTreeWalksTheFilesystem(t *testing.T) {
	sh := newShell()
	out := sh.run("tree")
	if len(out) != 1 || out[0].isErr {
		t.Fatalf("tree errored: %+v", out)
	}
	for _, want := range []string{"notes/", "experience.md", "directories"} {
		if !strings.Contains(out[0].text, want) {
			t.Errorf("tree output missing %q:\n%s", want, out[0].text)
		}
	}
}

func TestSeqIsBounded(t *testing.T) {
	// An unbounded seq would print millions of nodes into the DOM and lock the tab.
	sh := newShell()
	out := sh.run("seq 100000")
	if len(out) != 1 || out[0].isErr {
		t.Fatalf("seq errored: %+v", out)
	}
	if lines := len(splitLines(out[0].text)); lines > 1002 {
		t.Fatalf("seq printed %d lines; it must be capped", lines)
	}
}

func TestSleepRefusesToBlock(t *testing.T) {
	// Blocking here would freeze the browser's only thread and take the page down with it.
	sh := newShell()
	out := sh.run("sleep 5")
	if len(out) != 1 || !out[0].isErr {
		t.Fatalf("sleep should refuse: %+v", out)
	}
}

func TestDiffReportsDifferencesAndSilenceOnMatch(t *testing.T) {
	sh := newShell()
	sh.run("echo one > a.txt")
	sh.run("echo one > b.txt")
	if out := sh.run("diff a.txt b.txt"); len(out) != 0 {
		t.Fatalf("identical files should produce no output, got %+v", out)
	}
	sh.run("echo two > b.txt")
	out := sh.run("diff a.txt b.txt")
	if len(out) != 1 || out[0].isErr || !strings.Contains(out[0].text, "two") {
		t.Fatalf("diff should report the change: %+v", out)
	}
}

func TestCalendarGridStaysAligned(t *testing.T) {
	// Marking today with brackets shifted the rest of its row by a character, breaking the grid.
	out := calendar(time.Date(2026, 8, 9, 12, 0, 0, 0, time.UTC))
	lines := splitLines(out)
	if len(lines) < 4 {
		t.Fatalf("calendar too short:\n%s", out)
	}
	header := lines[1]
	if header != "Su Mo Tu We Th Fr Sa" {
		t.Fatalf("header = %q", header)
	}
	for _, l := range lines[2:] {
		if strings.HasPrefix(l, "today:") {
			continue
		}
		if len(l) > len(header) {
			t.Fatalf("week row wider than the header (%d > %d): %q", len(l), len(header), l)
		}
		// Every day cell must sit on a 3-column stride.
		for i := 2; i < len(l); i += 3 {
			if l[i] != ' ' {
				t.Fatalf("column %d of %q is not a separator — the grid has shifted", i, l)
			}
		}
	}
	if !strings.Contains(out, "today: Sunday 9 August") {
		t.Fatalf("calendar should name today:\n%s", out)
	}
}

func TestUnameDoesNotSayJs(t *testing.T) {
	out := unameOut(map[string]bool{"a": true})
	if strings.Contains(out, "Js ") {
		t.Fatalf("uname should not title-case GOOS into %q", out)
	}
	if !strings.HasPrefix(out, "Wasm ") {
		t.Fatalf("uname -a = %q; want it to lead with Wasm", out)
	}
}

func TestCorruptCachedTreeIsDiscarded(t *testing.T) {
	// A cached root that is not a directory cannot be repaired in place, and would leave the
	// terminal permanently empty with nothing on screen explaining why.
	js.Global().Get("localStorage").Call("setItem", vfsKey, `{"c":"not a tree"}`)
	defer js.Global().Get("localStorage").Call("removeItem", vfsKey)
	v := newVFS()
	if v.root == nil || !v.root.Dir {
		t.Fatal("a corrupt cached root should have been replaced by a fresh seed")
	}
	if v.get(v.resolve("notes/experience.md")) == nil {
		t.Fatal("the reseeded tree should carry the recruiter notes")
	}
}

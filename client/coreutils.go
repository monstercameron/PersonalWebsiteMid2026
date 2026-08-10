//go:build js && wasm

package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

// The second tier of the faux shell: the commands people actually try when they are testing whether
// a terminal is real. `uname` was the one that started this — a visitor typed it, the shell did not
// have it, and the did-you-mean helpfully suggested `anime`, which is exactly the moment a demo
// stops being convincing.
//
// The rule for everything here: implement it for real where the browser can (the virtual
// filesystem, the clock, the Go runtime's own heap, the machine the page is running on), and where
// it genuinely cannot, say so plainly rather than faking a plausible number. A terminal that lies
// about `free` is worse than one that admits it is a browser tab.

// coreutil runs one of the second-tier commands. It reports whether it handled the name, so the
// caller can fall through to portfolio programs and the did-you-mean suggestion.
func (s *shell) coreutil(name string, args []string, stdin string) (string, bool, error) {
	flags, ops := parseFlags(args)
	switch name {
	case "uname":
		return unameOut(flags), true, nil
	case "arch":
		return runtime.GOARCH + "\n", true, nil
	case "hostname":
		return "portfolio\n", true, nil
	case "id":
		return "uid=1000(cameron) gid=1000(cameron) groups=1000(cameron),27(sudo—allegedly)\n", true, nil
	case "groups":
		return "cameron sudo\n", true, nil
	case "users", "who", "w":
		return whoOut(), true, nil
	case "date":
		return time.Now().Format("Mon Jan  2 15:04:05 MST 2006") + "\n", true, nil
	case "cal":
		return calendar(time.Now()), true, nil
	case "free":
		return freeOut(), true, nil
	case "lscpu":
		return lscpuOut(), true, nil
	case "ps":
		return psOut(), true, nil
	case "env", "printenv":
		return envOut(), true, nil
	case "which", "whereis", "type":
		return whichOut(ops), true, nil
	case "tree":
		return s.tree(first(ops)), true, nil
	case "seq":
		return seqOut(ops)
	case "rev":
		return mapLines(s.readAll(ops, stdin), reverseString), true, nil
	case "tac":
		return tacOut(s.readAll(ops, stdin)), true, nil
	case "nl":
		return nlOut(s.readAll(ops, stdin)), true, nil
	case "tr":
		return trOut(ops, s.readAll(nil, stdin)), true, nil
	case "base64":
		// Read the decode switch off the raw args, not the parsed flags: "-d" is registered in
		// valueFlags for cut's delimiter, so parseFlags consumes it and its "value" and it never
		// reaches flags["d"]. Sharing one flag table across commands that disagree about a letter
		// is the trap here.
		return base64Out(hasFlag(args, "-d", "--decode"), s.readAll(ops, stdin))
	case "sha256sum", "md5sum":
		// One real hash rather than two: md5 is in the muscle memory, sha256 is the one worth
		// printing, and quietly labelling a sha256 digest as md5 would be a lie in a command whose
		// entire output is a claim about its input.
		return sha256Out(name, ops, s.readAll(ops, stdin)), true, nil
	case "xxd", "hexdump":
		return xxdOut(s.readAll(ops, stdin)), true, nil
	case "basename":
		return basenameOut(ops), true, nil
	case "dirname":
		return dirnameOut(ops), true, nil
	case "realpath":
		return strings.Join(s.fs.resolve(first(ops)), "/") + "\n", true, nil
	case "stat":
		return s.stat(first(ops))
	case "file":
		return s.fileType(first(ops))
	case "diff":
		return s.diff(ops)
	case "factor":
		return factorOut(ops), true, nil
	case "expr":
		return exprOut(ops), true, nil
	case "shuf":
		return shufOut(splitLines(s.readAll(ops, stdin))), true, nil
	case "yes":
		return yesOut(ops), true, nil
	case "banner", "figlet":
		return banner(strings.Join(ops, " ")), true, nil
	case "alias":
		return "alias ll='ls -l'\nalias la='ls -a'\nalias ..='cd ..'\n", true, nil
	case "sleep":
		// Sleeping here would block the browser's single thread and freeze the whole page — the
		// one thing a wasm app must never do.
		return "", true, fmt.Errorf("sleep: not blocking the event loop for you — this is a browser tab")
	case "wget":
		return "", true, fmt.Errorf("wget: network is sandboxed in the browser — the backend does the fetching")
	case "chmod", "chown", "sudoedit":
		return "", true, fmt.Errorf("%s: permissions are not modelled here — everything is yours already", name)
	case "exit", "logout", "quit":
		return "there is no exit. (close the tab, or press the red light.)\n", true, nil
	case "ping":
		return "", true, fmt.Errorf("ping: run it on its own — it measures a live round trip and cannot be piped")
	}
	return "", false, nil
}

// unameOut reports what this actually is. The honest answer is more interesting than a fake Linux
// string: the visitor is talking to a Go binary compiled to WebAssembly.
func unameOut(flags map[string]bool) string {
	// The kernel slot says "Wasm", not strings.Title(runtime.GOOS) — GOOS is "js", which titled to
	// "Js" and read like a typo rather than a fact.
	const kernel = "Wasm"
	if flags["a"] {
		return fmt.Sprintf("%s portfolio %s #1 SMP %s/%s GoWebComponents\n",
			kernel, runtime.Version(), runtime.GOOS, runtime.GOARCH)
	}
	return kernel + "\n"
}

// whoOut lists the session. There is exactly one visitor and it is whoever is reading.
func whoOut() string {
	return fmt.Sprintf("cameron  pts/0   %s  (you, right now)\n", time.Now().Format("2006-01-02 15:04"))
}

// freeOut reports the Go runtime's real heap. These are the actual numbers for the wasm instance
// rendering the page, read straight out of runtime.MemStats.
func freeOut() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	kb := func(b uint64) string { return fmt.Sprintf("%d", b/1024) }
	return "               total        used        free      (KiB, Go heap — the real numbers)\n" +
		fmt.Sprintf("Mem:     %10s  %10s  %10s\n", kb(m.HeapSys), kb(m.HeapAlloc), kb(m.HeapSys-m.HeapAlloc)) +
		fmt.Sprintf("GC:      %10d collections, %s total pause\n", m.NumGC, time.Duration(m.PauseTotalNs))
}

// lscpuOut reports what the browser will admit about the machine — deliberately coarse in every
// browser, so it is a hint rather than a fingerprint.
func lscpuOut() string {
	cores := "unknown"
	if n := js.Global().Get("navigator").Get("hardwareConcurrency"); n.Type() == js.TypeNumber {
		cores = strconv.Itoa(n.Int())
	}
	return "Architecture:        " + runtime.GOARCH + " (WebAssembly)\n" +
		"CPU(s):              " + cores + "   (as reported to the page)\n" +
		"Model name:          whatever you are reading this on\n" +
		"Runtime:             " + runtime.Version() + "\n"
}

// psOut lists the processes this page really is running.
func psOut() string {
	return "  PID TTY          TIME CMD\n" +
		"    1 pts/0    00:00:01 app.wasm\n" +
		"   14 pts/0    00:00:00 gwc-reconciler\n" +
		"   27 pts/0    00:00:00 grpctunnel\n" +
		"   41 pts/0    00:00:00 vfs\n"
}

// envOut prints an environment that describes the stack rather than inventing one.
func envOut() string {
	return strings.Join([]string{
		"HOME=/home/cam",
		"SHELL=/bin/gosh",
		"USER=cameron",
		"GOOS=" + runtime.GOOS,
		"GOARCH=" + runtime.GOARCH,
		"GOVERSION=" + runtime.Version(),
		"UI=GoWebComponents/v5",
		"TRANSPORT=gRPC-over-WebSocket",
		"TERM=xterm-256color",
	}, "\n") + "\n"
}

// whichOut answers where a command lives, for commands that exist.
func whichOut(ops []string) string {
	if len(ops) == 0 {
		return "usage: which <command>\n"
	}
	var b strings.Builder
	for _, name := range ops {
		found := false
		for _, c := range allCommands {
			if c == name {
				found = true
				break
			}
		}
		if found {
			fmt.Fprintf(&b, "/usr/bin/%s\n", name)
			continue
		}
		fmt.Fprintf(&b, "%s not found\n", name)
	}
	return b.String()
}

// tree renders the filesystem below path with box-drawing connectors.
func (s *shell) tree(path string) string {
	if path == "" {
		path = "."
	}
	segs := s.fs.resolve(path)
	n := s.fs.get(segs)
	if n == nil {
		return "tree: " + path + ": no such directory\n"
	}
	var b strings.Builder
	b.WriteString(path + "\n")
	dirs, files := 0, 0
	var walk func(node *fnode, prefix string)
	walk = func(node *fnode, prefix string) {
		if !node.Dir {
			return
		}
		names := childNames(node, false)
		for i, name := range names {
			connector, childPrefix := "├── ", prefix+"│   "
			if i == len(names)-1 {
				connector, childPrefix = "└── ", prefix+"    "
			}
			b.WriteString(prefix + connector + name + "\n")
			child := node.Children[strings.TrimSuffix(name, "/")]
			if child != nil && child.Dir {
				dirs++
				walk(child, childPrefix)
				continue
			}
			files++
		}
	}
	walk(n, "")
	fmt.Fprintf(&b, "\n%d directories, %d files\n", dirs, files)
	return b.String()
}

// seqOut prints a number sequence: seq LAST, seq FIRST LAST, or seq FIRST STEP LAST.
func seqOut(ops []string) (string, bool, error) {
	nums := make([]int, 0, len(ops))
	for _, o := range ops {
		v, err := strconv.Atoi(o)
		if err != nil {
			return "", true, fmt.Errorf("seq: invalid number: %s", o)
		}
		nums = append(nums, v)
	}
	first, step, last := 1, 1, 0
	switch len(nums) {
	case 1:
		last = nums[0]
	case 2:
		first, last = nums[0], nums[1]
	case 3:
		first, step, last = nums[0], nums[1], nums[2]
	default:
		return "", true, fmt.Errorf("usage: seq [first [step]] last")
	}
	if step == 0 {
		return "", true, fmt.Errorf("seq: step cannot be zero")
	}
	var b strings.Builder
	// Bounded so `seq 1 100000000` cannot lock the tab up printing into the DOM.
	const maxLines = 1000
	count := 0
	for v := first; (step > 0 && v <= last) || (step < 0 && v >= last); v += step {
		if count >= maxLines {
			b.WriteString("… (stopped at " + strconv.Itoa(maxLines) + " lines)\n")
			break
		}
		b.WriteString(strconv.Itoa(v) + "\n")
		count++
	}
	return b.String(), true, nil
}

// mapLines applies fn to every line of text.
func mapLines(text string, fn func(string) string) string {
	lines := splitLines(text)
	for i, l := range lines {
		lines[i] = fn(l)
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// reverseString reverses a string by runes, so multi-byte characters survive.
func reverseString(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// tacOut prints lines in reverse order.
func tacOut(text string) string {
	lines := splitLines(text)
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// nlOut numbers non-empty lines, the way nl does by default.
func nlOut(text string) string {
	var b strings.Builder
	n := 0
	for _, l := range splitLines(text) {
		if strings.TrimSpace(l) == "" {
			b.WriteString("\n")
			continue
		}
		n++
		fmt.Fprintf(&b, "%6d\t%s\n", n, l)
	}
	return b.String()
}

// trOut translates or deletes characters: tr SET1 SET2, or tr -d SET1.
func trOut(ops []string, text string) string {
	if len(ops) == 0 {
		return text
	}
	if len(ops) == 1 {
		return strings.Map(func(r rune) rune {
			if strings.ContainsRune(ops[0], r) {
				return -1
			}
			return r
		}, text)
	}
	from, to := []rune(ops[0]), []rune(ops[1])
	return strings.Map(func(r rune) rune {
		for i, f := range from {
			if f == r {
				if i < len(to) {
					return to[i]
				}
				return to[len(to)-1]
			}
		}
		return r
	}, text)
}

// hasFlag reports whether any of names appears literally in args.
func hasFlag(args []string, names ...string) bool {
	for _, a := range args {
		for _, n := range names {
			if a == n {
				return true
			}
		}
	}
	return false
}

// base64Out encodes, or decodes with -d.
func base64Out(decode bool, text string) (string, bool, error) {
	if decode {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(text))
		if err != nil {
			return "", true, fmt.Errorf("base64: invalid input")
		}
		return string(raw) + "\n", true, nil
	}
	return base64.StdEncoding.EncodeToString([]byte(text)) + "\n", true, nil
}

// sha256Out hashes the input. Called for md5sum too, which is answered honestly rather than
// silently mislabelled — see the dispatch comment.
func sha256Out(name string, ops []string, text string) string {
	sum := sha256.Sum256([]byte(text))
	label := "-"
	if len(ops) > 0 {
		label = ops[0]
	}
	out := hex.EncodeToString(sum[:]) + "  " + label + "\n"
	if name == "md5sum" {
		return out + "(that is a sha256 — md5 is not worth shipping, even as a joke)\n"
	}
	return out
}

// xxdOut renders a hex dump, capped so a large file cannot flood the scrollback.
func xxdOut(text string) string {
	const maxBytes = 512
	data := []byte(text)
	truncated := false
	if len(data) > maxBytes {
		data, truncated = data[:maxBytes], true
	}
	var b strings.Builder
	for off := 0; off < len(data); off += 16 {
		end := off + 16
		if end > len(data) {
			end = len(data)
		}
		chunk := data[off:end]
		fmt.Fprintf(&b, "%08x: ", off)
		for i := 0; i < 16; i++ {
			if i < len(chunk) {
				fmt.Fprintf(&b, "%02x", chunk[i])
			} else {
				b.WriteString("  ")
			}
			if i%2 == 1 {
				b.WriteString(" ")
			}
		}
		b.WriteString(" ")
		for _, c := range chunk {
			if c >= 32 && c < 127 {
				b.WriteByte(c)
				continue
			}
			b.WriteString(".")
		}
		b.WriteString("\n")
	}
	if truncated {
		b.WriteString("… (truncated at 512 bytes)\n")
	}
	return b.String()
}

// basenameOut strips the directory from a path.
func basenameOut(ops []string) string {
	p := first(ops)
	if i := strings.LastIndex(p, "/"); i >= 0 {
		p = p[i+1:]
	}
	return p + "\n"
}

// dirnameOut strips the final element from a path.
func dirnameOut(ops []string) string {
	p := first(ops)
	i := strings.LastIndex(p, "/")
	switch {
	case i > 0:
		return p[:i] + "\n"
	case i == 0:
		return "/\n"
	default:
		return ".\n"
	}
}

// stat reports what the virtual filesystem knows about a node.
func (s *shell) stat(path string) (string, bool, error) {
	if path == "" {
		return "", true, fmt.Errorf("usage: stat <path>")
	}
	n := s.fs.get(s.fs.resolve(path))
	if n == nil {
		return "", true, fmt.Errorf("stat: %s: no such file or directory", path)
	}
	kind, mode := "regular file", "-rw-r--r--"
	if n.Dir {
		kind, mode = "directory", "drwxr-xr-x"
	}
	return fmt.Sprintf("  File: %s\n  Size: %-10d Type: %s\nAccess: (%s)  Uid: (1000/cameron)\n Store: localStorage (this browser only)\n",
		path, sizeOf(n), kind, mode), true, nil
}

// fileType guesses a file's type from its name and content, as `file` does.
func (s *shell) fileType(path string) (string, bool, error) {
	if path == "" {
		return "", true, fmt.Errorf("usage: file <path>")
	}
	n := s.fs.get(s.fs.resolve(path))
	if n == nil {
		return "", true, fmt.Errorf("file: %s: no such file or directory", path)
	}
	switch {
	case n.Dir:
		return path + ": directory\n", true, nil
	case strings.HasSuffix(path, ".md"):
		return path + ": Markdown document, UTF-8 text\n", true, nil
	case strings.HasSuffix(path, ".pdf"):
		return path + ": PDF document (well — a note standing in for one)\n", true, nil
	case n.Content == "":
		return path + ": empty\n", true, nil
	default:
		return path + ": UTF-8 text\n", true, nil
	}
}

// diff reports the line differences between two files, in the unified-ish shape people expect.
func (s *shell) diff(ops []string) (string, bool, error) {
	if len(ops) < 2 {
		return "", true, fmt.Errorf("usage: diff <file1> <file2>")
	}
	read := func(p string) (string, error) {
		n := s.fs.get(s.fs.resolve(p))
		if n == nil || n.Dir {
			return "", fmt.Errorf("diff: %s: no such file", p)
		}
		return n.Content, nil
	}
	a, err := read(ops[0])
	if err != nil {
		return "", true, err
	}
	b, err := read(ops[1])
	if err != nil {
		return "", true, err
	}
	left, right := splitLines(a), splitLines(b)
	var out strings.Builder
	// A plain line-by-line comparison rather than a real LCS diff: this is a toy shell over a toy
	// filesystem, and an honest simple answer beats an approximate clever one.
	for i := 0; i < len(left) || i < len(right); i++ {
		switch {
		case i >= len(right):
			fmt.Fprintf(&out, "%d< %s\n", i+1, left[i])
		case i >= len(left):
			fmt.Fprintf(&out, "%d> %s\n", i+1, right[i])
		case left[i] != right[i]:
			fmt.Fprintf(&out, "%d< %s\n%d> %s\n", i+1, left[i], i+1, right[i])
		}
	}
	if out.Len() == 0 {
		return "", true, nil // identical: diff says nothing, like the real thing
	}
	return out.String(), true, nil
}

// factorOut prints the prime factorisation of each operand.
func factorOut(ops []string) string {
	var b strings.Builder
	for _, o := range ops {
		n, err := strconv.Atoi(o)
		if err != nil || n < 2 {
			fmt.Fprintf(&b, "factor: %s: not a number greater than 1\n", o)
			continue
		}
		fmt.Fprintf(&b, "%d:", n)
		for d := 2; d*d <= n; d++ {
			for n%d == 0 {
				fmt.Fprintf(&b, " %d", d)
				n /= d
			}
		}
		if n > 1 {
			fmt.Fprintf(&b, " %d", n)
		}
		b.WriteString("\n")
	}
	if b.Len() == 0 {
		return "usage: factor <number>...\n"
	}
	return b.String()
}

// exprOut evaluates a single binary integer expression, which is what `expr` is used for in
// practice. Anything more would be a calculator, and `bc` is not what anyone came here for.
func exprOut(ops []string) string {
	if len(ops) != 3 {
		return "usage: expr <a> <+|-|*|/|%> <b>\n"
	}
	a, err1 := strconv.Atoi(ops[0])
	b, err2 := strconv.Atoi(ops[2])
	if err1 != nil || err2 != nil {
		return "expr: non-integer argument\n"
	}
	switch ops[1] {
	case "+":
		return strconv.Itoa(a+b) + "\n"
	case "-":
		return strconv.Itoa(a-b) + "\n"
	case "*", "x":
		return strconv.Itoa(a*b) + "\n"
	case "/":
		if b == 0 {
			return "expr: division by zero\n"
		}
		return strconv.Itoa(a/b) + "\n"
	case "%":
		if b == 0 {
			return "expr: division by zero\n"
		}
		return strconv.Itoa(a%b) + "\n"
	default:
		return "expr: unknown operator " + ops[1] + "\n"
	}
}

// shufOut shuffles lines using the browser's RNG.
func shufOut(lines []string) string {
	rnd := js.Global().Get("Math")
	out := append([]string{}, lines...)
	for i := len(out) - 1; i > 0; i-- {
		j := int(rnd.Call("random").Float() * float64(i+1))
		if j > i {
			j = i
		}
		out[i], out[j] = out[j], out[i]
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n") + "\n"
}

// yesOut prints its argument — bounded, because the real one never stops and this one shares a
// thread with the page.
func yesOut(ops []string) string {
	text := "y"
	if len(ops) > 0 {
		text = strings.Join(ops, " ")
	}
	return strings.TrimRight(strings.Repeat(text+"\n", 20), "\n") +
		"\n… (the real one never stops; this one shares a thread with the page)\n"
}

// bannerGlyphs is a 5-row block font covering what a banner is actually used for.
var bannerGlyphs = map[rune][5]string{
	'A': {" ## ", "#  #", "####", "#  #", "#  #"},
	'B': {"### ", "#  #", "### ", "#  #", "### "},
	'C': {" ###", "#   ", "#   ", "#   ", " ###"},
	'D': {"### ", "#  #", "#  #", "#  #", "### "},
	'E': {"####", "#   ", "### ", "#   ", "####"},
	'F': {"####", "#   ", "### ", "#   ", "#   "},
	'G': {" ###", "#   ", "# ##", "#  #", " ###"},
	'H': {"#  #", "#  #", "####", "#  #", "#  #"},
	'I': {"###", " # ", " # ", " # ", "###"},
	'J': {"   #", "   #", "   #", "#  #", " ## "},
	'K': {"#  #", "# # ", "##  ", "# # ", "#  #"},
	'L': {"#   ", "#   ", "#   ", "#   ", "####"},
	'M': {"#   #", "## ##", "# # #", "#   #", "#   #"},
	'N': {"#  #", "## #", "# ##", "#  #", "#  #"},
	'O': {" ## ", "#  #", "#  #", "#  #", " ## "},
	'P': {"### ", "#  #", "### ", "#   ", "#   "},
	'Q': {" ## ", "#  #", "#  #", "# # ", " # #"},
	'R': {"### ", "#  #", "### ", "# # ", "#  #"},
	'S': {" ###", "#   ", " ## ", "   #", "### "},
	'T': {"#####", "  #  ", "  #  ", "  #  ", "  #  "},
	'U': {"#  #", "#  #", "#  #", "#  #", " ## "},
	'V': {"#   #", "#   #", "#   #", " # # ", "  #  "},
	'W': {"#   #", "#   #", "# # #", "## ##", "#   #"},
	'X': {"#  #", " ## ", " ## ", " ## ", "#  #"},
	'Y': {"#   #", " # # ", "  #  ", "  #  ", "  #  "},
	'Z': {"####", "   #", " ## ", "#   ", "####"},
	'0': {" ## ", "#  #", "#  #", "#  #", " ## "},
	'1': {" # ", "## ", " # ", " # ", "###"},
	'2': {" ## ", "#  #", "  # ", " #  ", "####"},
	'3': {"### ", "   #", " ## ", "   #", "### "},
	'4': {"#  #", "#  #", "####", "   #", "   #"},
	'5': {"####", "#   ", "### ", "   #", "### "},
	'6': {" ## ", "#   ", "### ", "#  #", " ## "},
	'7': {"####", "   #", "  # ", " #  ", " #  "},
	'8': {" ## ", "#  #", " ## ", "#  #", " ## "},
	'9': {" ## ", "#  #", " ###", "   #", " ## "},
	'!': {"#", "#", "#", " ", "#"},
	'?': {"### ", "   #", " ## ", "    ", " #  "},
	'.': {" ", " ", " ", " ", "#"},
	'-': {"    ", "    ", "####", "    ", "    "},
	' ': {"  ", "  ", "  ", "  ", "  "},
}

// banner renders text as block letters. Unknown characters are dropped rather than rendered as a
// gap, so a stray emoji does not blow a hole through the middle of a word.
func banner(text string) string {
	if text == "" {
		return "usage: banner <text>\n"
	}
	text = strings.ToUpper(text)
	// Bounded: a long line of block letters is unreadable anyway and would scroll forever.
	if len([]rune(text)) > 12 {
		text = string([]rune(text)[:12])
	}
	rows := [5]strings.Builder{}
	for _, r := range text {
		glyph, ok := bannerGlyphs[r]
		if !ok {
			continue
		}
		for i := 0; i < 5; i++ {
			rows[i].WriteString(glyph[i] + " ")
		}
	}
	var b strings.Builder
	for i := 0; i < 5; i++ {
		b.WriteString(strings.TrimRight(rows[i].String(), " ") + "\n")
	}
	return b.String()
}

// calendar renders the month containing t, marking today with brackets.
func calendar(t time.Time) string {
	first := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	daysInMonth := first.AddDate(0, 1, -1).Day()
	title := first.Format("January 2006")

	var b strings.Builder
	// Centre the title over the 20-column grid.
	pad := (20 - len(title)) / 2
	if pad < 0 {
		pad = 0
	}
	b.WriteString(strings.Repeat(" ", pad) + title + "\n")
	b.WriteString("Su Mo Tu We Th Fr Sa\n")
	b.WriteString(strings.Repeat("   ", int(first.Weekday())))
	for day := 1; day <= daysInMonth; day++ {
		// Every cell is exactly two columns wide. Bracketing today shifted the rest of its row by
		// a character and broke the grid, and the real cal marks the day with reverse video rather
		// than with extra characters — so the marker moved to its own line below.
		b.WriteString(fmt.Sprintf("%2d", day))
		weekday := (int(first.Weekday()) + day - 1) % 7
		if weekday == 6 {
			b.WriteString("\n")
			continue
		}
		b.WriteString(" ")
	}
	if !strings.HasSuffix(b.String(), "\n") {
		b.WriteString("\n")
	}
	b.WriteString("today: " + t.Format("Monday 2 January") + "\n")
	return b.String()
}

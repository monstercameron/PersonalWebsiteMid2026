//go:build js && wasm

package main

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

// splitTop splits on a separator (used for && and |). Simple, quote-unaware — fine for a toy shell.
func splitTop(s, sep string) []string { return strings.Split(s, sep) }

// parseCmd tokenizes a command string into a name and args, honoring single/double quotes.
func parseCmd(s string) (string, []string) {
	toks := tokenize(s)
	if len(toks) == 0 {
		return "", nil
	}
	return toks[0], toks[1:]
}

// tokenize splits on spaces but keeps quoted spans intact.
func tokenize(s string) []string {
	var toks []string
	var cur strings.Builder
	var quote rune
	flush := func() {
		if cur.Len() > 0 {
			toks = append(toks, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '\'' || r == '"':
			quote = r
		case r == ' ' || r == '\t':
			flush()
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return toks
}

// valueFlags take the following token as their value (skipped when collecting operands).
var valueFlags = map[string]bool{"-n": true, "-name": true, "-type": true, "-d": true, "-f": true}

// parseFlags splits args into boolean single-letter flags and positional operands. Flags that
// take a value (e.g. -n 10, -name "*.go") have their value skipped here; read it via flagVal.
func parseFlags(args []string) (map[string]bool, []string) {
	flags := map[string]bool{}
	var ops []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if len(a) > 1 && strings.HasPrefix(a, "-") {
			if valueFlags[a] {
				i++ // skip the value
				continue
			}
			for _, ch := range a[1:] {
				flags[string(ch)] = true
			}
			continue
		}
		ops = append(ops, a)
	}
	return flags, ops
}

// flagVal returns the token following flag, or "".
func flagVal(args []string, flag string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == flag {
			return args[i+1]
		}
	}
	return ""
}

func first(ops []string) string {
	if len(ops) > 0 {
		return ops[0]
	}
	return ""
}
func nth(ops []string, i int) string {
	if i < len(ops) {
		return ops[i]
	}
	return ""
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// wc counts lines (with -l) or lines/words/chars.
func wc(flags map[string]bool, text string) string {
	trimmed := strings.TrimRight(text, "\n")
	lines := 0
	if trimmed != "" {
		lines = strings.Count(trimmed, "\n") + 1
	}
	if flags["l"] {
		return fmt.Sprintf("%d\n", lines)
	}
	words := len(strings.Fields(text))
	return fmt.Sprintf("%d %d %d\n", lines, words, len(text))
}

// sortLines sorts stdin lines (lexicographic, or by human size with -h).
func sortLines(flags map[string]bool, text string) string {
	lines := splitLines(text)
	if flags["h"] {
		sort.SliceStable(lines, func(i, j int) bool { return parseHuman(lines[i]) < parseHuman(lines[j]) })
	} else {
		sort.Strings(lines)
	}
	return strings.Join(lines, "\n") + "\n"
}

// uniqLines removes adjacent duplicate lines.
func uniqLines(text string) string {
	var out []string
	prev := "\x00"
	for _, l := range splitLines(text) {
		if l != prev {
			out = append(out, l)
			prev = l
		}
	}
	return strings.Join(out, "\n") + "\n"
}

// cutCols extracts a field from each line by delimiter.
func cutCols(delim, fieldStr, text string) string {
	if delim == "" {
		delim = "\t"
	}
	f, _ := strconv.Atoi(fieldStr)
	if f < 1 {
		f = 1
	}
	var out []string
	for _, l := range splitLines(text) {
		cols := strings.Split(l, delim)
		if f <= len(cols) {
			out = append(out, cols[f-1])
		} else {
			out = append(out, "")
		}
	}
	return strings.Join(out, "\n") + "\n"
}

// sedSub applies a single s/PAT/REP/[g] substitution to each line.
func sedSub(expr, text string) string {
	if !strings.HasPrefix(expr, "s") || len(expr) < 4 {
		return text
	}
	sep := expr[1]
	parts := strings.Split(expr[2:], string(sep))
	if len(parts) < 2 {
		return text
	}
	pat, rep := parts[0], parts[1]
	global := len(parts) >= 3 && strings.Contains(parts[2], "g")
	n := 1
	if global {
		n = -1
	}
	var out []string
	for _, l := range splitLines(text) {
		out = append(out, strings.Replace(l, pat, rep, n))
	}
	return strings.Join(out, "\n") + "\n"
}

// awkPrint handles the common awk '{print $N}' form.
func awkPrint(expr, text string) string {
	col := 0
	if i := strings.Index(expr, "$"); i >= 0 {
		numStr := ""
		for _, r := range expr[i+1:] {
			if r >= '0' && r <= '9' {
				numStr += string(r)
			} else {
				break
			}
		}
		col, _ = strconv.Atoi(numStr)
	}
	var out []string
	for _, l := range splitLines(text) {
		fields := strings.Fields(l)
		if col == 0 {
			out = append(out, l)
		} else if col <= len(fields) {
			out = append(out, fields[col-1])
		} else {
			out = append(out, "")
		}
	}
	return strings.Join(out, "\n") + "\n"
}

// splitLines splits text into lines, dropping a single trailing newline.
func splitLines(text string) []string {
	t := strings.TrimRight(text, "\n")
	if t == "" {
		return nil
	}
	return strings.Split(t, "\n")
}

// human formats a byte count like du -h.
func human(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1fM", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1fK", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%dB", n)
	}
}

// parseHuman parses a leading human size (e.g. "3.1M") into bytes for sorting.
func parseHuman(s string) float64 {
	s = strings.TrimSpace(s)
	num := ""
	unit := 1.0
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' {
			num += string(r)
			continue
		}
		switch r {
		case 'K', 'k':
			unit = 1 << 10
		case 'M', 'm':
			unit = 1 << 20
		case 'G', 'g':
			unit = 1 << 30
		}
		break
	}
	f, _ := strconv.ParseFloat(num, 64)
	return f * unit
}

// globMatch reports whether name matches a shell glob like *.go.
func globMatch(pattern, name string) bool {
	ok, err := path.Match(pattern, name)
	return err == nil && ok
}

// manPage returns a short faux man page for a command.
func manPage(cmd string) string {
	pages := map[string]string{
		"ls":   "LS(1)\n  ls [-lah] [path] — list directory contents.",
		"cd":   "CD(1)\n  cd [path] — change the working directory.",
		"grep": "GREP(1)\n  grep [-Rni] pattern [path] — search text.",
		"find": "FIND(1)\n  find [path] [-name glob] [-type f|d] — walk the tree.",
		"echo": "ECHO(1)\n  echo text [> file] — print text (or write it to a file).",
		"cat":  "CAT(1)\n  cat file... — print file contents.",
	}
	if p, ok := pages[cmd]; ok {
		return p + "\n"
	}
	if cmd == "" {
		return "What manual page do you want? (try `man ls`)\n"
	}
	return fmt.Sprintf("No manual entry for %s (this is a faux shell — try `help`)\n", cmd)
}

// allCommands is the completion vocabulary (portfolio programs + shell commands).
var allCommands = []string{
	"help", "about", "whoami", "projects", "open", "neofetch", "links", "contact", "clear",
	"pwd", "ls", "cd", "mkdir", "touch", "cp", "mv", "rm", "cat", "less", "head", "tail", "nano",
	"echo", "grep", "find", "sort", "uniq", "cut", "sed", "awk", "wc", "du", "df", "history",
	"curl", "ssh", "tar", "man", "reset",
	// Added by the 2026-08-09 pass. The easter eggs are deliberately absent — they complete on
	// Tab only if you already know the name, which is the whole point of an easter egg.
	"resume", "anime", "tour", "share", "theme", "stats", "uptime", "bench",
	// The second tier: the commands people reach for when they are testing whether a terminal is
	// real. `uname` earned this list its existence — a visitor typed it, the shell did not have
	// it, and the did-you-mean suggested `anime`.
	"uname", "arch", "hostname", "id", "groups", "users", "who", "w", "date", "cal", "free",
	"lscpu", "ps", "env", "printenv", "which", "whereis", "type", "tree", "seq", "rev", "tac",
	"nl", "tr", "base64", "sha256sum", "md5sum", "xxd", "hexdump", "basename", "dirname",
	"realpath", "stat", "file", "diff", "factor", "expr", "shuf", "yes", "banner", "figlet",
	"alias", "sleep", "wget", "chmod", "chown", "exit", "logout", "quit", "ping",
}

// complete performs Ubuntu-style Tab completion of the last token: a command name for the first
// token, otherwise a path in the relevant directory. It completes to the longest common prefix,
// finishing fully (with a trailing space) when there's a single match.
func complete(inputStr string, sh *shell) string {
	if strings.TrimSpace(inputStr) == "" {
		return inputStr
	}
	toks := strings.Split(inputStr, " ")
	last := toks[len(toks)-1]

	var candidates []string
	if len(toks) == 1 {
		for _, c := range allCommands {
			if strings.HasPrefix(c, last) {
				candidates = append(candidates, c)
			}
		}
	} else if sh != nil {
		dirPart, basePart := splitPath(last)
		segs := sh.fs.resolve(dirPart)
		if n := sh.fs.get(segs); n != nil && n.Dir {
			for name, child := range n.Children {
				if strings.HasPrefix(name, basePart) {
					full := name
					if child.Dir {
						full += "/"
					}
					candidates = append(candidates, joinPath(dirPart, full))
				}
			}
		}
	}
	if len(candidates) == 0 {
		return inputStr
	}
	completed := longestCommonPrefix(candidates)
	if len(candidates) == 1 {
		completed = candidates[0]
		if !strings.HasSuffix(completed, "/") {
			completed += " "
		}
	}
	toks[len(toks)-1] = completed
	return strings.Join(toks, " ")
}

// splitPath splits a path into its directory portion and final base for completion.
func splitPath(p string) (dir, base string) {
	i := strings.LastIndex(p, "/")
	if i < 0 {
		return "", p
	}
	return p[:i], p[i+1:]
}

// joinPath rejoins a directory portion with a completed name.
func joinPath(dir, name string) string {
	if dir == "" {
		return name
	}
	return dir + "/" + name
}

// longestCommonPrefix returns the longest common prefix of the candidate strings.
func longestCommonPrefix(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	prefix := ss[0]
	for _, s := range ss[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

// --- virtual filesystem mutators ---

// mkdir creates a directory.
func (v *vfs) mkdir(p string) error {
	parent, name := v.parentOf(p)
	if parent == nil || !parent.Dir {
		return fmt.Errorf("mkdir: %s: no such parent", p)
	}
	if parent.Children == nil {
		parent.Children = map[string]*fnode{}
	}
	if _, ok := parent.Children[name]; ok {
		return fmt.Errorf("mkdir: %s: already exists", p)
	}
	parent.Children[name] = dir(map[string]*fnode{})
	return nil
}

// touch creates an empty file if it doesn't exist.
func (v *vfs) touch(p string) error {
	parent, name := v.parentOf(p)
	if parent == nil || !parent.Dir {
		return fmt.Errorf("touch: %s: no such parent", p)
	}
	if parent.Children == nil {
		parent.Children = map[string]*fnode{}
	}
	if _, ok := parent.Children[name]; !ok {
		parent.Children[name] = file("")
	}
	return nil
}

// writeFile creates or overwrites a file (used by > redirection).
func (v *vfs) writeFile(p, content string) error {
	parent, name := v.parentOf(p)
	if parent == nil || !parent.Dir {
		return fmt.Errorf("cannot write %s", p)
	}
	if parent.Children == nil {
		parent.Children = map[string]*fnode{}
	}
	parent.Children[name] = file(content)
	return nil
}

// rm removes a file or (with recursive) a directory.
func (v *vfs) rm(p string, recursive bool) error {
	parent, name := v.parentOf(p)
	if parent == nil || parent.Children == nil {
		return fmt.Errorf("rm: %s: not found", p)
	}
	n, ok := parent.Children[name]
	if !ok {
		return fmt.Errorf("rm: %s: not found", p)
	}
	if n.Dir && !recursive && len(n.Children) > 0 {
		return fmt.Errorf("rm: %s: is a directory (use -r)", p)
	}
	delete(parent.Children, name)
	return nil
}

// destFor resolves the real destination path for cp/mv.
//
// When dst names an existing directory, the source goes INSIDE it under its own base name, which is
// what cp and mv actually do. Without this, `cp notes/README.md notes` resolved to parent=/home/cam,
// name="notes" and replaced the whole notes/ directory with a single file — silently destroying
// every recruiter-facing document in one command, with no error and no way back short of `reset`.
func (v *vfs) destFor(src, dst string) string {
	if n := v.get(v.resolve(dst)); n != nil && n.Dir {
		return strings.TrimSuffix(dst, "/") + "/" + baseName(src)
	}
	return dst
}

// baseName returns the final element of a path.
func baseName(p string) string {
	p = strings.TrimSuffix(p, "/")
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

// samePath reports whether two paths resolve to the same node.
func (v *vfs) samePath(a, b string) bool {
	return strings.Join(v.resolve(a), "/") == strings.Join(v.resolve(b), "/")
}

// cp copies a node. Copying onto an existing directory places the copy inside it.
func (v *vfs) cp(src, dst string) error {
	sn := v.get(v.resolve(src))
	if sn == nil {
		return fmt.Errorf("cp: %s: not found", src)
	}
	dst = v.destFor(src, dst)
	if v.samePath(src, dst) {
		return fmt.Errorf("cp: %s and %s are the same file", src, dst)
	}
	parent, name := v.parentOf(dst)
	if parent == nil || !parent.Dir {
		return fmt.Errorf("cp: %s: bad destination", dst)
	}
	if parent.Children == nil {
		parent.Children = map[string]*fnode{}
	}
	parent.Children[name] = cloneNode(sn)
	v.save()
	return nil
}

// mv moves/renames a node.
//
// The same-path guard is not a nicety: mv was implemented as copy-then-remove, so `mv x x` copied a
// node over itself and then deleted it — losing the file entirely, where the real mv does nothing.
func (v *vfs) mv(src, dst string) error {
	dst = v.destFor(src, dst)
	if v.samePath(src, dst) {
		return nil // real mv treats this as a no-op, not as a delete
	}
	if err := v.cp(src, dst); err != nil {
		return err
	}
	if err := v.rm(src, true); err != nil {
		return err
	}
	v.save()
	return nil
}

// cloneNode deep-copies a node.
func cloneNode(n *fnode) *fnode {
	if n == nil {
		return nil
	}
	c := &fnode{Dir: n.Dir, Content: n.Content}
	if n.Children != nil {
		c.Children = map[string]*fnode{}
		for k, ch := range n.Children {
			c.Children[k] = cloneNode(ch)
		}
	}
	return c
}

// walk visits a subtree rooted at segs, calling fn for each node with its absolute path.
func (v *vfs) walk(segs []string, fn func(path string, n *fnode)) {
	start := v.get(segs)
	if start == nil {
		return
	}
	var rec func(segs []string, n *fnode)
	rec = func(segs []string, n *fnode) {
		p := "/" + strings.Join(segs, "/")
		if len(segs) == 0 {
			p = "/"
		}
		fn(p, n)
		if n.Dir {
			names := make([]string, 0, len(n.Children))
			for k := range n.Children {
				names = append(names, k)
			}
			sort.Strings(names)
			for _, k := range names {
				rec(append(append([]string{}, segs...), k), n.Children[k])
			}
		}
	}
	rec(segs, start)
}

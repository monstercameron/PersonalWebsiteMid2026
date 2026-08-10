//go:build js && wasm

package main

import (
	"fmt"
	"strings"
)

// shell is a faux Unix shell over the virtual filesystem. It supports pipes (|), sequential
// chaining (&&), and output redirection (>), plus ~30 common commands.
type shell struct {
	fs *vfs
	// nav is the command history plus the Up/Down browsing cursor. It records every command the
	// terminal runs, portfolio programs included — recording it inside run() would have missed
	// them, since those never reach the shell.
	nav histNav
}

// newShell builds a shell backed by the (localStorage-cached) virtual filesystem.
func newShell() *shell { return &shell{fs: newVFS()} }

// prompt returns the current working directory shown in the prompt (~ for home).
func (s *shell) prompt() string {
	p := s.fs.pwd()
	if strings.HasPrefix(p, "/home/cam") {
		return "~" + strings.TrimPrefix(p, "/home/cam")
	}
	return p
}

// run executes a full command line and returns rendered output nodes.
func (s *shell) run(line string) []outLine {
	var out []outLine
	for _, seg := range splitTop(line, "&&") {
		text, err := s.pipeline(strings.TrimSpace(seg))
		if err != nil {
			out = append(out, outLine{text: err.Error(), isErr: true})
			break // stop the && chain on error
		}
		if text != "" {
			out = append(out, outLine{text: text})
		}
	}
	return out
}

// outLine is one block of shell output (text may contain newlines; rendered pre-wrap).
type outLine struct {
	text  string
	isErr bool
}

// pipeline runs a single | pipeline (one && segment), threading stdout into the next stdin.
func (s *shell) pipeline(seg string) (string, error) {
	if seg == "" {
		return "", nil
	}
	stdin := ""
	for _, part := range splitTop(seg, "|") {
		part = strings.TrimSpace(part)
		cmdStr, redirect := part, ""
		if i := strings.Index(part, ">"); i >= 0 {
			cmdStr = strings.TrimSpace(part[:i])
			redirect = strings.TrimSpace(part[i+1:])
		}
		name, args := parseCmd(cmdStr)
		if name == "" {
			continue
		}
		out, err := s.exec(name, args, stdin)
		if err != nil {
			return "", err
		}
		if redirect != "" {
			if err := s.fs.writeFile(redirect, out); err != nil {
				return "", err
			}
			s.fs.save()
			out = ""
		}
		stdin = out
	}
	return stdin, nil
}

// exec runs one command with args and stdin, returning stdout.
func (s *shell) exec(name string, args []string, stdin string) (string, error) {
	flags, ops := parseFlags(args)
	switch name {
	case "pwd":
		return s.fs.pwd() + "\n", nil
	case "ls":
		return s.ls(flags, ops)
	case "cd":
		return "", s.cd(first(ops))
	case "mkdir":
		return "", s.each(ops, func(p string) error { return s.fs.mkdir(p) })
	case "touch":
		return "", s.each(ops, func(p string) error { return s.fs.touch(p) })
	case "rm":
		return "", s.each(ops, func(p string) error { return s.fs.rm(p, flags["r"] || flags["R"]) })
	case "cp":
		return "", s.fs.cp(nth(ops, 0), nth(ops, 1))
	case "mv":
		return "", s.fs.mv(nth(ops, 0), nth(ops, 1))
	case "cat", "less":
		return s.cat(ops, stdin)
	case "head":
		return s.headTail(true, flags, ops, stdin)
	case "tail":
		return s.headTail(false, flags, ops, stdin)
	case "echo":
		return strings.Join(ops, " ") + "\n", nil
	case "wc":
		return wc(flags, s.readAll(ops, stdin)), nil
	case "sort":
		return sortLines(flags, s.readAll(ops, stdin)), nil
	case "uniq":
		return uniqLines(s.readAll(ops, stdin)), nil
	case "cut":
		return cutCols(flagVal(args, "-d"), flagVal(args, "-f"), s.readAll(ops, stdin)), nil
	case "grep":
		return s.grep(args, stdin)
	case "find":
		return s.find(args)
	case "sed":
		return sedSub(first(ops), s.readAll(ops[minInt(1, len(ops)):], stdin)), nil
	case "awk":
		return awkPrint(first(ops), s.readAll(ops[minInt(1, len(ops)):], stdin)), nil
	case "du":
		return s.du(flags, ops), nil
	case "df":
		return "Filesystem   Size  Used  Avail  Use%\nvfs           64M   3.1M   61M    5%\n", nil
	case "history":
		return s.hist(), nil
	case "nano":
		return "", fmt.Errorf("nano: no interactive editor here — use `echo \"text\" > %s`", first(ops))
	case "curl":
		return "", fmt.Errorf("curl: network is sandboxed in the browser — the backend does the fetching")
	case "ssh":
		return "", fmt.Errorf("ssh: connection to %s refused (nice try 😏)", first(ops))
	case "tar":
		return "tar: archived (faux) — the virtual fs doesn't leave your browser\n", nil
	case "man":
		return manPage(first(ops)), nil
	case "reset":
		s.fs.reset()
		return "filesystem restored to its original state.\n", nil
	default:
		// The second tier of coreutils (uname, date, cal, tree, base64, …) lives in coreutils.go
		// to keep this switch readable. It reports whether it handled the name.
		if out, handled, err := s.coreutil(name, args, stdin); handled {
			return out, err
		}
		// A portfolio program reached through the shell — i.e. inside a pipeline or a redirect —
		// renders as plain text from the same row definitions the styled version uses. This is what
		// makes `projects | grep Go` and `about > notes/cam.md` work instead of quietly doing
		// nothing.
		if isPortfolio(name) {
			return rowsText(programOutput(name, args)), nil
		}
		if guess := didYouMean(name); guess != "" {
			return "", fmt.Errorf("%s: command not found — did you mean `%s`?", name, guess)
		}
		return "", fmt.Errorf("%s: command not found — try `help`", name)
	}
}

// --- filesystem-reading command helpers ---

// ls lists a directory (or the cwd).
func (s *shell) ls(flags map[string]bool, ops []string) (string, error) {
	path := first(ops)
	n := s.fs.get(s.fs.resolve(path))
	if n == nil {
		return "", fmt.Errorf("ls: %s: no such file or directory", path)
	}
	if !n.Dir {
		return path + "\n", nil
	}
	names := childNames(n, flags["a"])
	if flags["l"] {
		var b strings.Builder
		for _, name := range names {
			child := n.Children[strings.TrimSuffix(name, "/")]
			kind := "-"
			if strings.HasSuffix(name, "/") {
				kind = "d"
			}
			fmt.Fprintf(&b, "%srw-r--r--  %5d  %s\n", kind, sizeOf(child), name)
		}
		return b.String(), nil
	}
	return strings.Join(names, "  ") + "\n", nil
}

// cd changes the working directory.
func (s *shell) cd(path string) error {
	if path == "" {
		path = "~"
	}
	segs := s.fs.resolve(path)
	n := s.fs.get(segs)
	if n == nil || !n.Dir {
		return fmt.Errorf("cd: %s: no such directory", path)
	}
	s.fs.cwd = segs
	return nil
}

// cat concatenates files (or echoes stdin when no files given).
func (s *shell) cat(ops []string, stdin string) (string, error) {
	if len(ops) == 0 {
		return stdin, nil
	}
	var b strings.Builder
	for _, p := range ops {
		n := s.fs.get(s.fs.resolve(p))
		if n == nil || n.Dir {
			return "", fmt.Errorf("cat: %s: no such file", p)
		}
		b.WriteString(n.Content)
	}
	return b.String(), nil
}

// headTail returns the first/last n lines.
func (s *shell) headTail(head bool, flags map[string]bool, ops []string, stdin string) (string, error) {
	_ = flags
	n := 10
	// -n N is parsed as a flag value pair
	text := s.readAll(ops, stdin)
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n") + "\n", nil
	}
	if head {
		lines = lines[:n]
	} else {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n") + "\n", nil
}

// grep searches stdin or files (recursively with -R).
func (s *shell) grep(args []string, stdin string) (string, error) {
	flags, ops := parseFlags(args)
	if len(ops) == 0 {
		return "", fmt.Errorf("usage: grep [-Rni] <pattern> [path]")
	}
	pat := ops[0]
	ci := flags["i"]
	needle := pat
	if ci {
		needle = strings.ToLower(pat)
	}
	match := func(line string) bool {
		if ci {
			return strings.Contains(strings.ToLower(line), needle)
		}
		return strings.Contains(line, pat)
	}
	var b strings.Builder
	emit := func(prefix, line string, num int) {
		if flags["n"] {
			fmt.Fprintf(&b, "%s%d:%s\n", prefix, num, line)
		} else {
			fmt.Fprintf(&b, "%s%s\n", prefix, line)
		}
	}
	if len(ops) >= 2 && (flags["R"] || flags["r"]) {
		s.fs.walk(s.fs.resolve(ops[1]), func(path string, n *fnode) {
			if n.Dir {
				return
			}
			for i, line := range strings.Split(n.Content, "\n") {
				if match(line) {
					emit(path+":", line, i+1)
				}
			}
		})
		return b.String(), nil
	}
	src := stdin
	if len(ops) >= 2 {
		n := s.fs.get(s.fs.resolve(ops[1]))
		if n != nil && !n.Dir {
			src = n.Content
		}
	}
	for i, line := range strings.Split(strings.TrimRight(src, "\n"), "\n") {
		if match(line) {
			emit("", line, i+1)
		}
	}
	return b.String(), nil
}

// find lists paths under a root, optionally filtered by -name / -type.
func (s *shell) find(args []string) (string, error) {
	flags, ops := parseFlags(args)
	_ = flags
	root := "."
	if len(ops) > 0 {
		root = ops[0]
	}
	nameGlob := flagVal(args, "-name")
	typ := flagVal(args, "-type")
	var b strings.Builder
	s.fs.walk(s.fs.resolve(root), func(path string, n *fnode) {
		if typ == "f" && n.Dir {
			return
		}
		if typ == "d" && !n.Dir {
			return
		}
		base := path[strings.LastIndex(path, "/")+1:]
		if nameGlob != "" && !globMatch(nameGlob, base) {
			return
		}
		b.WriteString(path + "\n")
	})
	return b.String(), nil
}

// du reports sizes.
func (s *shell) du(flags map[string]bool, ops []string) string {
	_ = flags
	targets := ops
	if len(targets) == 0 {
		targets = []string{"."}
	}
	// expand a trailing "*" into the cwd children
	if len(targets) == 1 && targets[0] == "*" {
		targets = nil
		if n := s.fs.get(s.fs.cwd); n != nil && n.Dir {
			for _, name := range childNames(n, false) {
				targets = append(targets, strings.TrimSuffix(name, "/"))
			}
		}
	}
	var b strings.Builder
	for _, t := range targets {
		n := s.fs.get(s.fs.resolve(t))
		fmt.Fprintf(&b, "%-6s %s\n", human(sizeOf(n)), t)
	}
	return b.String()
}

// hist prints the command history.
func (s *shell) hist() string {
	var b strings.Builder
	for i, h := range s.nav.entries {
		fmt.Fprintf(&b, "%4d  %s\n", i+1, h)
	}
	return b.String()
}

// readAll returns the concatenated content of the file operands, or stdin if none.
func (s *shell) readAll(ops []string, stdin string) string {
	if len(ops) == 0 {
		return stdin
	}
	var b strings.Builder
	for _, p := range ops {
		if n := s.fs.get(s.fs.resolve(p)); n != nil && !n.Dir {
			b.WriteString(n.Content)
		}
	}
	return b.String()
}

// each runs fn over each operand, saving the fs on success.
func (s *shell) each(ops []string, fn func(string) error) error {
	if len(ops) == 0 {
		return fmt.Errorf("missing operand")
	}
	for _, p := range ops {
		if err := fn(p); err != nil {
			return err
		}
	}
	s.fs.save()
	return nil
}

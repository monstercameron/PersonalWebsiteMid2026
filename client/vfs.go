//go:build js && wasm

package main

import (
	"encoding/json"
	"sort"
	"strings"
	"syscall/js"

	"github.com/monstercameron/earlcameron/internal/notes"
)

// fnode is a virtual filesystem node: a directory (Children) or a file (Content).
type fnode struct {
	Dir      bool              `json:"d,omitempty"`
	Content  string            `json:"c,omitempty"`
	Children map[string]*fnode `json:"ch,omitempty"`
}

// vfs is an in-memory virtual filesystem that caches itself in localStorage so it survives
// reloads. Paths are resolved against the current working directory (cwd).
type vfs struct {
	root *fnode
	cwd  []string
}

const vfsKey = "earlcameron.vfs.v2"

// newVFS loads the cached filesystem, or seeds a fresh one, and starts in the home directory.
func newVFS() *vfs {
	v := &vfs{cwd: []string{"home", "cam"}}
	// A cached tree whose root is not a directory cannot be healed by repairSeed — it only patches
	// inside directories — and would leave the terminal permanently empty for that visitor, with
	// `reset` as the only way out and nothing on screen suggesting it. Treat it as no cache at all.
	if !v.load() || v.root == nil || !v.root.Dir {
		v.root = seedTree()
		v.save()
		return v
	}
	// The cached tree is the visitor's sandbox and is kept as they left it — except that a
	// visitor who ran `rm -rf ~/notes` would otherwise have destroyed the recruiter-facing
	// content permanently, for every later visit, with nothing on screen saying so. Anything
	// seeded and now missing comes back; anything they created or edited is left alone.
	if repairSeed(v.root, seedTree()) {
		v.save()
	}
	return v
}

// reset restores the seeded filesystem and returns to the home directory, discarding whatever the
// visitor did to it. This is the in-session escape hatch that `repairSeed` is for across reloads.
func (v *vfs) reset() {
	v.root = seedTree()
	v.cwd = []string{"home", "cam"}
	v.save()
}

// repairSeed re-adds nodes present in seed but missing from cached, recursively, and reports
// whether it changed anything. Existing nodes are never overwritten: an edited seed file keeps the
// visitor's edit, and a seeded path they replaced with a different node kind is left as theirs.
func repairSeed(cached, seed *fnode) bool {
	if cached == nil || seed == nil || !cached.Dir || !seed.Dir {
		return false
	}
	changed := false
	for name, seedChild := range seed.Children {
		existing, ok := cached.Children[name]
		if !ok {
			if cached.Children == nil {
				cached.Children = map[string]*fnode{}
			}
			cached.Children[name] = cloneNode(seedChild)
			changed = true
			continue
		}
		if repairSeed(existing, seedChild) {
			changed = true
		}
	}
	return changed
}

// dir makes a directory node from the given children.
func dir(children map[string]*fnode) *fnode { return &fnode{Dir: true, Children: children} }

// file makes a file node with the given content.
func file(content string) *fnode { return &fnode{Content: content} }

// seedTree builds the initial /home/cam filesystem, including recruiter-facing notes.
func seedTree() *fnode {
	return dir(map[string]*fnode{
		"home": dir(map[string]*fnode{
			"cam": dir(map[string]*fnode{
				"about.md":   file(notes.About),
				"resume.pdf": file("%PDF-1.4 (faux) — run `resume` on the site to download the real one.\n"),
				// The three headliners, matching internal/content.featured's leading entries.
				"projects": dir(map[string]*fnode{
					"cashflux.md":    file("CashFlux — local-first budgeting, all Go/WASM, no JS framework.\n"),
					"articleflux.md": file("ArticleFlux — self-hosted feed reader, Go all the way down. Real gRPC in the browser.\n"),
					"gwc.md":         file("GoWebComponents — a React-style UI framework in Go→WASM. This site runs on it.\n"),
				}),
				"notes": dir(map[string]*fnode{
					"README.md":        file(notes.README),
					"experience.md":    file(notes.Experience),
					"skills.md":        file(notes.Skills),
					"projects.md":      file(notes.Projects),
					"working-style.md": file(notes.WorkingStyle),
				}),
				".secrets": file("nice try 😏\n"),
			}),
		}),
	})
}

// load restores the filesystem from localStorage. It returns false if nothing was cached.
func (v *vfs) load() bool {
	raw := js.Global().Get("localStorage").Call("getItem", vfsKey)
	if !raw.Truthy() {
		return false
	}
	var root fnode
	if err := json.Unmarshal([]byte(raw.String()), &root); err != nil {
		return false
	}
	v.root = &root
	return true
}

// save persists the filesystem to localStorage.
func (v *vfs) save() {
	b, err := json.Marshal(v.root)
	if err != nil {
		return
	}
	js.Global().Get("localStorage").Call("setItem", vfsKey, string(b))
}

// pwd returns the current working directory as an absolute path.
func (v *vfs) pwd() string {
	if len(v.cwd) == 0 {
		return "/"
	}
	return "/" + strings.Join(v.cwd, "/")
}

// resolve turns a path (absolute, relative, ~, ., ..) into absolute segments.
func (v *vfs) resolve(path string) []string {
	var segs []string
	switch {
	case path == "" || path == "~":
		if path == "~" {
			return []string{"home", "cam"}
		}
		return append([]string{}, v.cwd...)
	case strings.HasPrefix(path, "~/"):
		segs = []string{"home", "cam"}
		path = strings.TrimPrefix(path, "~/")
	case strings.HasPrefix(path, "/"):
		path = strings.TrimPrefix(path, "/")
	default:
		segs = append([]string{}, v.cwd...)
	}
	for _, p := range strings.Split(path, "/") {
		switch p {
		case "", ".":
		case "..":
			if len(segs) > 0 {
				segs = segs[:len(segs)-1]
			}
		default:
			segs = append(segs, p)
		}
	}
	return segs
}

// get returns the node at the given segments, or nil.
func (v *vfs) get(segs []string) *fnode {
	n := v.root
	for _, s := range segs {
		if n == nil || !n.Dir || n.Children == nil {
			return nil
		}
		n = n.Children[s]
	}
	return n
}

// parentOf returns the parent node and the final name for a path.
func (v *vfs) parentOf(path string) (*fnode, string) {
	segs := v.resolve(path)
	if len(segs) == 0 {
		return nil, ""
	}
	return v.get(segs[:len(segs)-1]), segs[len(segs)-1]
}

// childNames returns the sorted child names of a directory node (dirs marked with a trailing /).
func childNames(n *fnode, all bool) []string {
	var names []string
	for name, c := range n.Children {
		if !all && strings.HasPrefix(name, ".") {
			continue
		}
		if c.Dir {
			name += "/"
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sizeOf returns a node's total content size in bytes (recursive for dirs).
func sizeOf(n *fnode) int {
	if n == nil {
		return 0
	}
	if !n.Dir {
		return len(n.Content)
	}
	total := 0
	for _, c := range n.Children {
		total += sizeOf(c)
	}
	return total
}

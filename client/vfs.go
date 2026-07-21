//go:build js && wasm

package main

import (
	"encoding/json"
	"sort"
	"strings"
	"syscall/js"
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

const vfsKey = "earlcameron.vfs.v1"

// newVFS loads the cached filesystem, or seeds a fresh one, and starts in the home directory.
func newVFS() *vfs {
	v := &vfs{cwd: []string{"home", "cam"}}
	if !v.load() {
		v.root = seedTree()
		v.save()
	}
	return v
}

// dir makes a directory node from the given children.
func dir(children map[string]*fnode) *fnode { return &fnode{Dir: true, Children: children} }

// file makes a file node with the given content.
func file(content string) *fnode { return &fnode{Content: content} }

// seedTree builds the initial /home/cam filesystem.
func seedTree() *fnode {
	return dir(map[string]*fnode{
		"home": dir(map[string]*fnode{
			"cam": dir(map[string]*fnode{
				"about.md":   file("AI-native systems engineer. I build ambitious things fast by pairing systems judgment with LLM leverage.\n"),
				"resume.pdf": file("%PDF-1.4 (faux) — run `resume` on the site to download the real one.\n"),
				"projects": dir(map[string]*fnode{
					"gwc.md":        file("GoWebComponents — a React-style UI framework in Go→WASM. This site runs on it.\n"),
					"cashflux.md":   file("CashFlux — local-first budgeting, all Go/WASM, no JS framework.\n"),
					"wasibrowser.md": file("WASIBrowser — a no-JavaScript browser that renders WebAssembly apps.\n"),
				}),
				"notes": dir(map[string]*fnode{
					"todo.txt": file("TODO: ship the terminal\nTODO: wire gRPC programs\ndone: match the mockup\nTODO: i18n pass\n"),
					"ideas.txt": file("agent-native assembly\non-device LLMs\na browser with no javascript\n"),
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

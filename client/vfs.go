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

const vfsKey = "earlcameron.vfs.v2"

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

// seedTree builds the initial /home/cam filesystem, including recruiter-facing notes.
func seedTree() *fnode {
	return dir(map[string]*fnode{
		"home": dir(map[string]*fnode{
			"cam": dir(map[string]*fnode{
				"about.md":   file(aboutMD),
				"resume.pdf": file("%PDF-1.4 (faux) — run `resume` on the site to download the real one.\n"),
				"projects": dir(map[string]*fnode{
					"gwc.md":         file("GoWebComponents — a React-style UI framework in Go→WASM. This site runs on it.\n"),
					"cashflux.md":    file("CashFlux — local-first budgeting, all Go/WASM, no JS framework.\n"),
					"wasibrowser.md": file("WASIBrowser — a no-JavaScript browser that renders WebAssembly apps.\n"),
				}),
				"notes": dir(map[string]*fnode{
					"README.md":        file(notesReadme),
					"experience.md":    file(experienceMD),
					"skills.md":        file(skillsMD),
					"projects.md":      file(projectsMD),
					"working-style.md": file(workingStyleMD),
				}),
				".secrets": file("nice try 😏\n"),
			}),
		}),
	})
}

// --- recruiter-facing notes (professional, tech-fit content only) ---

const aboutMD = `Earl Cameron — AI-native systems engineer · Lauderhill, FL

Senior software engineer (UKG, since 2020). I build production systems across the stack — Go,
C#/.NET on the backend; React, Angular, TypeScript on the front — and I ship fast by pairing deep
systems judgment with LLMs in the loop. Recently focused on agentic systems and AI infrastructure.

Curious recruiter? ` + "`ls notes`" + ` and read on.
`

const notesReadme = `notes/ — a quick briefing for recruiters.

  cat notes/experience.md      roles, impact, education
  cat notes/skills.md          languages, frameworks, infra, AI/ML
  cat notes/projects.md        open-source work (github.com/monstercameron)
  cat notes/working-style.md   how I operate

Everything here is public and professional. The rest of the terminal is a playground —
type ` + "`help`" + ` or ` + "`projects`" + `.
`

const experienceMD = `EXPERIENCE

UKG — Software Engineer (P3 / Senior)                                    2020 – present
  HCM domain, and more recently an agentic-systems / AI-infrastructure org (agents & AI tooling).
  Selected work:
    · Angular → React modernization (React, Tremor, Node, MS SQL Server)
    · Unified Search micro-frontend
    · "48 Hours" ChatAssistant
    · Bryte ChangeJob proof-of-concept
    · HCM Pillar Dashboard
    · Go store/send benchmarks; client-per-core capacity experiments
    · Provider registries; authentication & session systems
    · SQL performance work (CXPACKET / SOS_SCHEDULER_YIELD); throughput & priority stored procs

EDUCATION
  B.S. Information Technology — Florida International University (FIU)
  A.S. Information Technology — Miami Dade College
`

const skillsMD = `SKILLS

  Languages   Go · C# · TypeScript / JavaScript · Python   (Rust, C — exploratory)
  Frontend    React · Angular · Blazor / Razor · Tailwind · WebAssembly (Go→WASM)
  Backend     .NET · Node · Flask · gRPC · REST
  Data        MS SQL Server · MySQL · MongoDB · SQLite
  Infra       Docker · IIS · WSL2 · GCP · DigitalOcean · Nginx
  AI / ML     OpenAI & Anthropic APIs · LangChain · FAISS · Whisper · Stable Diffusion ·
              local LLM runtimes · on-device inference (Snapdragon NPU, QNN, ONNX Runtime GenAI,
              INT4 quantization)
  Workflow    AI-native — Claude Code, Copilot, Cursor; heavy but measured LLM use
`

const projectsMD = `SELECTED OPEN-SOURCE WORK   github.com/monstercameron

  CashFlux            Local-first budgeting suite in Go/WASM — 40+ pages, rules engine, AI layer.
  ArticleFlux         Self-hosted feed reader, Go all the way down — FTS5 search, TTS, multi-tenant.
  GoWebComponents     A React-style UI framework in Go→WASM. This site runs on it.
  GoGRPCBridge        gRPC over WebSockets for the browser — no proxy. Carries this site's traffic.
  WASIBrowser         A no-JavaScript browser that renders WebAssembly apps via a custom ABI.
  SemanticScript      An agent-first programming language — auditable source designed for LLMs.
  SemanticAssembly    Agent-native RISC-V assembly layer.
  SemanticPortrait    Privacy-first journaling → a living self-portrait graph via a local model.
  Snapdragon LLMs     Running 12–27B models on the Snapdragon X2 NPU/GPU (QNN / ONNX pipelines).
`

const workingStyleMD = `HOW I WORK

  · Establish the mechanism first, challenge weak assumptions, refine the architecture, then spec.
  · Direct and technically substantive — comfortable down at low-level architecture.
  · Honest about feasibility and toolchain limits.
  · Design sense: dark, polished, engineered-intentional UIs; clean information hierarchy;
    privacy-first; explainable, reversible operations.
`

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

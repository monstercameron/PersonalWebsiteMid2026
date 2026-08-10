package content

// The Flux project pages.
//
// These are the six products that carry the "Flux" brand, each with its own commissioned artwork
// (web/static/brand). Each gets a dedicated page at /projects/<slug> — the case-study surface
// TODOS §14.F asks for, in the shape §14.G specifies: what it is in one jargon-free sentence,
// what it does, what it is built on, what was hard, and where the evidence is.
//
// Two rules govern the copy in this file, and both matter more than the prose:
//
//  1. **Every factual claim is sourced from the project's own repository** — its README, its
//     tests, its measurements. Numbers are quoted, never estimated. If a project's README does
//     not support a sentence here, the sentence does not ship.
//  2. **Maturity is stated, not implied.** Most are alpha, prototype or (for AnimeFeedFlux) not
//     yet written; they are built in spare time with AI agents in the loop, and each page says
//     so in its own section rather than burying it. A recruiter who discovers that themselves
//     discounts everything else on the page; one who is told it up front reads the rest as credible.
//
// What is deliberately NOT here: first-person motivation ("why I built it") and authorship
// claims. TODOS §14.F reserves those for Cam's own words — they are judgment claims and must
// not be synthesised. The pages are built so those paragraphs can be dropped in later.

// FluxProject is the full content of one project page.
type FluxProject struct {
	// Slug is the route segment, the brand-asset filename prefix, and the theme.Brands key —
	// one identifier so a page, its art and its palette cannot drift apart.
	Slug string
	// Name is the product name; Head and Tail split it at the brand lockup's two-tone break
	// ("Article" + "Flux"), reproducing the poster's wordmark in the site's own typeface.
	Name, Head, Tail string
	// Tagline is the product's own line, taken from its poster art.
	Tagline string
	// Status is the short maturity chip ("alpha", "prototype", "v1.0.0").
	Status string
	// Lede is the one jargon-free sentence a non-specialist can repeat (§14.G question 1).
	Lede string
	// Pitch is the sales paragraph: what a person gets, in their terms.
	Pitch string
	// Repo and Demo are the outbound links. Demo is empty when there is nothing runnable.
	Repo, Demo string
	// DemoLabel names what Demo actually is, so nobody has to guess whether it runs.
	DemoLabel string
	// Capabilities are what it does (§14.G question 2).
	Capabilities []Capability
	// Stack is what it is built on (§14.G question 3) — the underlying libraries, named and
	// linked, including the ones Cam wrote.
	Stack []StackItem
	// Hard are the two or three genuinely difficult problems (§14.G question 4). These are the
	// paragraphs a hiring manager reads.
	Hard []Problem
	// Evidence are countable facts pulled from the repository (§14.G question 5).
	Evidence []Fact
	// Maturity is this project's honest status paragraph — what works, what does not.
	Maturity string
	// Art reports whether this project has the commissioned brand set in web/static/brand
	// (`<slug>-mark.webp`, `-poster.webp`, `-og.jpg`). The page hard-references those files by
	// slug, so a project without them would render broken images here, on every other Flux page's
	// "more projects" strip, and in the card a link unfurler builds. When false the page falls back
	// to a typographic identity instead — see site.projectHero and site.brandPoster.
	//
	// It is a field rather than a lookup against the filesystem because the pages are rendered once
	// at startup and a missing asset must be a compile-time-visible decision, not a runtime 404.
	// TestFluxArtAssetsExist holds it to the filesystem so it cannot become a claim.
	Art bool
	// PosterDark marks a brand sheet drawn on a dark ground rather than the near-white the other
	// posters assume. It selects the plate the poster is mounted on — see site.brandPlate.
	PosterDark bool
	// PosterNote overrides the caption under the brand sheet. The default asserts that the product
	// is real and the code is linked above, which is not true of a project that is still only a
	// specification — see site.posterNote.
	PosterNote string
}

// Capability is one thing the product does, in the user's vocabulary.
type Capability struct{ Title, Body string }

// StackItem is one underlying technology, with a link and the reason it is there.
type StackItem struct {
	Name string
	// Href links the technology; empty for things with no useful destination.
	Href string
	// Mine marks the pieces Cam wrote himself — the ones worth a recruiter's attention.
	Mine bool
	// Why is one line on the job this piece does in this project.
	Why string
}

// Problem is one hard engineering problem and how it was resolved.
type Problem struct{ Title, Body string }

// Fact is a countable figure and its label.
type Fact struct{ Figure, Label string }

// disclosure is the shared honesty block shown on every Flux page.
//
// It is one paragraph, placed where it will be read rather than in a footnote, and it is written
// to be an asset rather than an apology: shipping this much surface area alone, in spare time,
// with agents in the loop, is the actual claim being made. The projects are the evidence for it.
const disclosure = "These are personal projects, built on nights and weekends with AI agents in " +
	"the loop — that is how one person ships this much surface area at once. Most of them are " +
	"early: alpha or prototype, with polish that varies a lot by area. What is not early is the " +
	"architecture, the test suites, and the measurements — every number on this page is counted " +
	"from the repository and every claim is one you can check against the code."

// Disclosure returns the shared spare-time / AI-assisted / maturity statement.
func Disclosure() string { return disclosure }

// FluxProjects returns the Flux project pages in display order.
func FluxProjects() []FluxProject {
	const gh = "https://github.com/monstercameron/"
	return []FluxProject{
		{
			Slug: "cashflux", Name: "CashFlux", Head: "Cash", Tail: "Flux",
			Art:     true,
			Tagline: "See where your money moves.",
			Status:  "alpha · live",
			Lede:    "A budgeting app that runs entirely inside your browser tab and keeps your financial data on your device.",
			Pitch: "No account to create, nothing uploaded, no server holding your transaction history. " +
				"Open it and it works — accounts, transactions, budgets, goals, bills, reports and a " +
				"financial to-do list — and it keeps working on a plane. The whole application, including " +
				"a real SQLite database, is a single Go program the browser downloads once and then runs.",
			Repo: gh + "CashFlux", Demo: "https://monstercameron.github.io/CashFlux/", DemoLabel: "Try it in your browser",
			Capabilities: []Capability{
				{"Your data stays yours", "It runs local-first against a SQLite database inside the tab. The one network call is an AI insight, and only when you click it, with your own key."},
				{"Budgeting that fits how you already think", "Zero-based or envelope budgeting, snowball or avalanche debt payoff, cash-flow forecasting, and cleared-versus-uncleared reconciliation."},
				{"Built for a household, not just a person", "Ownership is explicit — shared or per-member — so two people can run one set of books without guessing whose expense is whose."},
				{"Every figure shows its work", "A budget, a forecast, an allocation score: each traces back to the transactions behind it. Money is stored as integer cents with an explicit currency, never a float."},
			},
			Stack: []StackItem{
				{"GoWebComponents", gh + "GoWebComponents", true, "My Go→WASM UI framework — components, hooks, router and the typed CSS engine that replaces the stylesheet."},
				{"GoGRPCBridge", gh + "GoGRPCBridge", true, "Carries optional multi-device sync as real gRPC over one WebSocket, with no REST layer in between."},
				{"ncruces/go-sqlite3", "https://github.com/ncruces/go-sqlite3", false, "A real SQLite engine compiled into the client — no C, no cgo, no server."},
				{"Go → WebAssembly", "", false, "The UI, the styling, the money math and the database are one language in one binary."},
			},
			Hard: []Problem{
				{"A database in a browser tab, not a fake one", "Local-first is easy to claim and hard to mean. CashFlux runs an actual SQLite engine client-side, which makes the query layer, the migrations and the reconciliation logic the same code you would write on a server — and makes offline the default rather than a degraded mode."},
				{"Money that survives arithmetic", "Balances, budgets, debt payoff and a sandboxed formula engine are pure, dependency-free Go packages under table-driven tests. Currency is integer cents plus an explicit code, because the alternative is a rounding bug nobody finds until it is in somebody's savings goal."},
				{"Sync without giving up local-first", "Multi-device sync is opt-in and additive: the app is complete without a backend, and turning sync on adds an encrypted server-side store over the gRPC tunnel instead of relocating the source of truth."},
			},
			Evidence: []Fact{
				{"221", "internal packages"}, {"2,998", "tests"}, {"50", "application routes"},
				{"26", "documented releases"}, {"6", "weeks, first commit to platform"},
			},
			Maturity: "Alpha, and labelled that way in its own README. The architecture and the pure-logic " +
				"core are real and heavily tested, the live app is usable day to day, and the feature list " +
				"describes what is in the app rather than a roadmap — but maturity varies by area and some " +
				"surfaces are still rough.",
		},
		{
			Slug: "articleflux", Name: "ArticleFlux", Head: "Article", Tail: "Flux",
			Art:     true,
			Tagline: "Signal, ranked for you.",
			Status:  "shipping · self-hosted",
			Lede:    "A feed reader you host yourself, with the three-pane layout and keyboard shortcuts Google Reader taught everyone.",
			Pitch: "Google Reader died in 2013 and the replacements were a choice between somebody else's " +
				"server holding your reading history and a self-hosted PHP app from 2011. This is the third " +
				"option: one binary, one SQLite file, no runtime dependencies, and a client that behaves " +
				"like an application instead of a page.",
			Repo: gh + "ArticleFlux", Demo: "https://monstercameron.github.io/ArticleFlux/", DemoLabel: "Try it in your browser",
			Capabilities: []Capability{
				{"The key map your hands already know", "j, k, o, s, r and u do what you expect. The vocabulary was kept on purpose, so muscle memory transfers on day one."},
				{"Firehose scale without the stutter", "The development instance runs 153 feeds and 7,303 articles in a 126 MiB SQLite file, through a virtualised list that does not care how long it gets."},
				{"Search, tags, notes and text-to-speech", "Full-text search over SQLite FTS5, per-source hues, tagging, notes, and a reader that will read to you."},
				{"One poll per feed, no matter how many readers", "Multi-tenant and deduplicated: a popular feed is fetched once for everybody subscribed to it."},
			},
			Stack: []StackItem{
				{"GoWebComponents", gh + "GoWebComponents", true, "My Go→WASM framework. There is not one .css file and not one application .js file in the repository, and CI fails if either appears."},
				{"GoGRPCBridge", gh + "GoGRPCBridge", true, "My gRPC-over-WebSocket tunnel. Sixty RPCs across five services, including the streaming ones that push a new article to the screen without a refresh."},
				{"SQLite + FTS5", "https://www.sqlite.org/fts5.html", false, "The whole datastore, search index included, in one file you can copy."},
				{"Protocol Buffers", "https://protobuf.dev/", false, "proto/articleflux/v1 is the only contract; client and server are both generated from it, so they cannot disagree about a field name."},
			},
			Hard: []Problem{
				{"Real gRPC in a browser, with no proxy", "A standard gRPC client cannot open a socket from a js/wasm build, and the usual surrender is a hand-rolled REST layer that drifts from the schema. GoGRPCBridge tunnels HTTP/2 gRPC frames over a WebSocket instead — all four RPC types, deadlines, metadata and interceptors intact — so the .proto stays the single source of truth on both sides."},
				{"A list that stays fast at firehose scale", "Thousands of unread items is the normal case, not the stress test. The list is virtualised and the read state is designed around it, so marking all read with undo stays instant at 5,913 unread."},
				{"The cost of deleting the toolchain, stated out loud", "Writing the client in Go means a 32.9 MB wasm bundle, roughly 7 MB gzipped, cached after first load. CI fails the build if it grows more than 5% without someone bumping the baseline on purpose. It is a real trade and the README argues it rather than hiding it."},
			},
			Evidence: []Fact{
				{"153", "feeds in the dev instance"}, {"7,303", "articles indexed"},
				{"60", "gRPC methods"}, {"0", "application .js files"}, {"0", ".css files"},
			},
			Maturity: "The most finished of the five: it runs live, it is used daily, and the public demo is " +
				"the shipping client with only the transport swapped underneath — same components, same " +
				"generated stubs. It is still a single-maintainer, spare-time project, and the demo cannot " +
				"do anything that needs a server.",
		},
		{
			Slug: "pixelflux", Name: "PixelFlux", Head: "Pixel", Tail: "Flux",
			Art:     true,
			Tagline: "Find meaning in every photo.",
			Status:  "prototype · Windows",
			Lede:    "Search your own photographs by what is in them, on a laptop, with nothing leaving the machine.",
			Pitch: "Type \"red car\" and you get the red car — not the file named car.jpg, and not a cloud " +
				"service's answer. A vision model has already looked at every photograph and written a " +
				"paragraph about it, a segmenter has outlined what is in it, a detector has found the " +
				"faces, and all of that is folded into one searchable vector. The only time it touches " +
				"the network is the first-run model download, because you pressed a button.",
			Repo: gh + "PixelFlux", DemoLabel: "",
			Capabilities: []Capability{
				{"Ask in plain words", "Search runs over meaning, not filenames or tags. Nothing has to be labelled first."},
				{"It reads your library once, patiently", "Describe, segment, detect and embed run as a queue over hours rather than a button you wait on, and the result is cached forever."},
				{"Faces grouped, names yours", "Grouping is by appearance and is recomputed rather than stored. Only the names you type are treated as facts."},
				{"Private by construction, not by policy", "Browsing, analysis and search make no network requests at all, and the interface's content security policy forbids them."},
			},
			Stack: []StackItem{
				{".NET 10 · MAUI Blazor Hybrid", "https://dotnet.microsoft.com/", false, "A native Windows desktop app with a web-technology interface, one codebase."},
				{"ONNX Runtime + DirectML", "https://onnxruntime.ai/", false, "Runs the vision, embedding, detection and recognition models on-device, with a per-model choice of accelerator."},
				{"CLIP · YuNet · SFace", "", false, "Image and text embeddings for search; face detection and recognition for people."},
				{"SQLite", "https://www.sqlite.org/", false, "The library index and the vector store, in a file on your disk."},
				{"Snapdragon X2 Elite (arm64)", "", false, "The build and every measurement on this page come from an arm64 laptop, fanless and thermally honest."},
			},
			Hard: []Problem{
				{"A caption in the search vector beats a better image encoder", "Blending the CLIP text embedding of the generated description with the image embedding, weighted 80/20 toward the image, took \"red car\" from 0.80 to 1.00 precision-at-five. Both extremes lose. The weight is part of the model version string, because otherwise changing it silently reuses the old vectors — which happened, and looked exactly like the change having no effect."},
				{"The bottleneck was never the model", "Face detection cost 114 ms a photograph, of which the network was under 10 ms; the rest was decoding a four-megapixel JPEG twice and discarding most of it. Decoding at the size the model actually wants took the face stage from 82 s to 25 s across the library, at a measured cost of 0.8% of detections."},
				{"A repetition penalty trades looping for hallucination, and that is the wrong trade", "Greedy decoding loops at the tail of a description; a penalty stops it and starts inventing specific strings — a sign, a box label, a name tag. For text that becomes a search index, a repeated sentence is harmless and an invented proper noun makes a photo findable by a word that is not in it. The decoder stops on a repeated sentence instead."},
			},
			Evidence: []Fact{
				{"207", "tests"}, {"11 s", "to describe one photograph"},
				{"0.80→1.00", "precision@5 after the caption blend"}, {"82→25 s", "face stage, after the decode fix"},
			},
			Maturity: "A prototype, and its README lists the limits without being asked: Windows only, arm64 " +
				"is the tested target, the vision model is small and sometimes wrong, face grouping is " +
				"appearance rather than identity, and there is no sync. The measurements are real and " +
				"reproducible; the product around them is early.",
		},
		{
			Slug: "codeflux", Name: "CodeFlux", Head: "Code", Tail: "Flux",
			Art:     true,
			Tagline: "Verified atoms. Better software.",
			Status:  "prototype · experimental",
			Lede:    "A coding agent that assembles a program out of small, separately verified pieces instead of writing it in one pass and hoping.",
			Pitch: "Most coding agents write a program and then try to check it. CodeFlux builds one from the " +
				"bottom up out of atoms — reusable units with a typed signature, a contract, declared " +
				"effects and the evidence that they work. Atoms compose into molecules, molecules into " +
				"control flow, control flow into a program, and every layer names the pieces that " +
				"discharge its guarantees.",
			Repo: gh + "CodeFlux", DemoLabel: "",
			Capabilities: []Capability{
				{"You read the plan before anything runs", "It proposes scope, steps and the checks it intends to satisfy. You describe an outcome in your own words; no specification required."},
				{"Verified work is meant to become project capital", "The design: a proven atom is recalled rather than rebuilt, and re-verified against the new run's contract, because reuse without that would inherit the old blind spot. Registration works; recall across runs has not fired yet, and the repository tracks that as an open ticket rather than a feature."},
				{"Authority comes from what an action is", "Permission is derived from the tool and its declared effects, not from what the model says it needs. A poisoned file can persuade a model to propose something; it cannot make it authorized."},
				{"Known, ambiguous and recommended stay separate", "A forecast is a range, an unreported price stays unknown instead of becoming zero, and a passing check means those checks passed and nothing more."},
			},
			Stack: []StackItem{
				{"Go", "https://go.dev", false, "One executable that installs per user and never asks for administrator rights — an agent requesting elevation is asking for far more trust than it needs."},
				{"gRPC + Protocol Buffers", "https://grpc.io", false, "The contract between the engine and its interface."},
				{"OS credential store", "", false, "Provider credentials are read from standard input, never from an argument every process can see, and stored by the operating system."},
				{"CodeQL + a two-branch CI gate", "", false, "A main gate and a dev pass run on every change; security analysis is part of the pipeline, not an audit afterwards."},
			},
			Hard: []Problem{
				{"Making authority structural instead of conversational", "Prompt injection stops being a category of attack when permission is not something a model can talk its way into. Authority is a property of the action — the tool, its ordered arguments, its declared effects — so the worst a poisoned file achieves is a proposal you see and decline."},
				{"Re-verification, not just reuse", "The cheap version of atom reuse is a cache. The correct version re-derives tests from the new run's contract and makes the atom earn its place again, which is what keeps a reused component from carrying an old assumption into a new context."},
				{"Writing down the kill criterion in advance", "The project is explicitly a bet — on functional decomposition into pure atoms, and on verified reuse compounding into lower cost. The plan states what result would kill it: no measurable improvement in defects, review time or total cost. Whether it holds is not settled, and the README says so."},
			},
			Evidence: []Fact{
				{"41", "stages in a run"}, {"11", "of them a reused atom skips"},
				{"3", "platforms — Windows, macOS, Linux"},
				{"0", "administrator rights required"},
			},
			Maturity: "A prototype, and the least finished of the five. It exists to test a hypothesis rather " +
				"than to be adopted, and the README states the hypothesis, the two bets underneath it, and " +
				"the condition that would end the project.",
		},
		{
			Slug: "schemaflux", Name: "SchemaFlux", Head: "Schema", Tail: "Flux",
			Art:     true,
			Tagline: "Structured AI that ships.",
			Status:  "v1.1.0 · stable API",
			Lede:    "A Go library that makes a language model return a typed Go value instead of a wall of text you have to parse.",
			Pitch: "One public API of fluent builders over one execution path. Application code reads like " +
				"Go — Extracting[Person](text).Strict().Run() — while retries, structured-output " +
				"contracts, logging, metrics and cost tracking stay in one place instead of being " +
				"re-implemented at every call site.",
			Repo: gh + "SchemaFlux", DemoLabel: "",
			Capabilities: []Capability{
				{"Typed results, not string wrangling", "Extract, transform, generate, classify, score, rank, cluster and compare — each returns the Go type you asked for, with generics doing the work."},
				{"Model claims and measured facts stay separate", "The result envelope keeps what the model asserted apart from what the library actually observed, so a confident answer never gets promoted to a verified one by accident."},
				{"Cost is a first-class output", "Retries, token accounting and spend tracking are centralized, so the price of a call is something you read rather than something you discover on the invoice."},
				{"An API surface that cannot drift quietly", "The public API is captured in a snapshot test. Adding or removing anything fails the build, which makes every change a reviewed decision."},
			},
			Stack: []StackItem{
				{"Go generics", "https://go.dev", false, "The builders are typed end to end — the return type is the type you named, not an any you assert."},
				{"OpenAI-compatible providers", "", false, "OpenAI is the default; the provider layer is where a different backend plugs in."},
				{"Architecture decision records", "", false, "The judgment calls, including the decision to ship 1.0 with known gaps, are written down as ADRs in the repository."},
			},
			Hard: []Problem{
				{"Not letting a pattern matcher pose as a classifier", "Redaction matches whole field names and validates card numbers with Luhn, so FirstName and APIKey are caught while Filename and a 16-digit order number are not. A bare nine-digit number is deliberately not treated as a social security number, because it is indistinguishable from an order ID. The README says plainly that it is a safety net under the fields you tag, not a substitute for tagging them."},
				{"Naming the cost of the convenient operation", "Semantic deduplication asks the model about pairs, which is O(n²) calls in the worst case. The documentation leads with that rather than burying it, because the failure mode is a bill, not an exception."},
				{"Shipping 1.0 with twelve unmet criteria, on the record", "Twelve of the project's own thirty-two acceptance criteria were not met when 1.0 shipped. They were listed with a reason each, and ADR 0005 argues why the version went out anyway. Writing down what a release does not do is a more useful artifact than a 0.9 that never ends — and it is why the gaps got closed in the open, across the ten commits that became 1.1.0."},
			},
			Evidence: []Fact{
				{"v1.1.0", "released, API-surface test guarding it"}, {"94.2%", "coverage floor, ratcheted"},
				{"1", "execution path behind every builder"},
			},
			Maturity: "The exception on this page: a library at a released 1.1.0 with an API-surface test guarding " +
				"it, and every behaviour claim in its README backed by a test. It is still honest about the " +
				"gaps — read \"What 1.0 does not include\" before depending on it for anything load-bearing.",
		},
		{
			// The newest of the six, and the only one with no product behind it yet. Its page is
			// built from its specification rather than its code, and says so in every place a
			// reader might otherwise assume otherwise — the status chip, the evidence figures and
			// the maturity paragraph.
			//
			// Its sheet is the one drawn on a dark ground rather than near-white, so PosterDark
			// swaps the plate underneath it. The wordmark splits "Anime" / "FeedFlux" to match the
			// lockup, where "Anime" is white and "Feed"+"Flux" carry the blue-to-violet gradient.
			Slug: "animefeedflux", Name: "AnimeFeedFlux", Head: "Anime", Tail: "FeedFlux",
			Art: true, PosterDark: true,
			Tagline: "Feeds that write themselves.",
			Status:  "planning · spec of record",
			Lede:    "A feed generator: describe the feed you want in a sentence, and it publishes on a schedule to any reader you already use.",
			Pitch: "A recipe carries a prompt, a schedule, a model and a budget. It runs on time, writes the " +
				"items, and publishes them at a stable address as RSS, Atom and JSON Feed — so the thing " +
				"you subscribe to is an ordinary feed, and nothing downstream has to know a model wrote " +
				"it. The whole product surface is one sentence: a URL that returns valid XML and never lies.",
			Repo: gh + "AnimeFeedFlux", DemoLabel: "",
			PosterNote: "The specification is real and linked above; the poster is a design exercise, not a claim.",
			Capabilities: []Capability{
				{"Describe a feed, get a URL", "A recipe is a prompt, a schedule, a model and a spend cap. What comes out is an ordinary feed address that Slack, or any reader, can subscribe to without knowing a model wrote it."},
				{"Two kinds of item, separated by design", "Generative items — trivia, on-this-day — are written by the model, which is the source. Grounded items — news, releases — are only edited by it; the facts and the links come from the publisher. Conflating the two is named in the plan as the main way the project fails, so they are split at the architecture level rather than by convention."},
				{"Corrections, not silent edits", "A reader that has already seen an item never re-fetches it, so editing one reaches nobody. Corrections are published as new items instead, which is the only version a subscriber will actually receive."},
				{"Anime-native, not a general feed bot", "The recipes it ships with are the ones an anime audience actually wants: a daily trivia question, the day's news ranked by impact, and a seasonal roundup of what is worth watching."},
			},
			Stack: []StackItem{
				{"SchemaFlux", gh + "SchemaFlux", true, "My typed-LLM library — the model returns a Go value, so a malformed generation is a type error rather than a parsing bug in the feed."},
				{"GoWebComponents", gh + "GoWebComponents", true, "The admin UI, Go compiled to WebAssembly — and deliberately the last phase, after the engine is already serving feeds."},
				{"GoGRPCBridge", gh + "GoGRPCBridge", true, "Real gRPC from the browser for the control plane; there is no REST/JSON API at all."},
				{"Go · SQLite with FTS5 · nginx", "", false, "One droplet, one writer. Scaling out means a second instance with its own database, not a cluster — which the plan states rather than discovers."},
			},
			Hard: []Problem{
				{"Making a hallucinated link unpublishable rather than unlikely", "The obvious failure of an AI-written news feed is a confident link to a page that does not exist, and prompting is not a defence against it. Sources are fetched first, their URLs normalised once, and only that candidate set is ever shown to the model; a link may be published only if it is byte-equal to a URL that was actually retrieved. An invented URL is not improbable — there is no code path that can emit one."},
				{"Slack's RSS app is stricter than the specification, and fails silently", "It does not error on a feed it dislikes; it simply stops posting. It needs a date on every item, items in sequence, and no duplicate timestamps — so a news run publishing three items at once would have had two dropped with no signal. It also bookmarks the newest date it has seen, which makes a backdated item invisible forever. That produced real constraints: strictly increasing timestamps enforced by a database constraint, a no-backdating rule, plain-text descriptions with the rich HTML in content:encoded, and trivia answers kept out of the description so a channel preview cannot spoil the question."},
				{"Putting the security boundary between two planes instead of inside one", "The publish plane is plain HTTPS, GET and HEAD only, unauthenticated, and holds a read-only database handle — a bug in the code path serving the public internet has no writer available to corrupt anything with. Every mutation is a gRPC call on a separate, authenticated, IP-allowlisted control plane. The split is the defence; the authentication is only the lock on the half that can write."},
			},
			Evidence: []Fact{
				{"3", "feed formats from one recipe — RSS, Atom, JSON Feed"},
				{"0", "invented links publishable, by construction"},
				{"2", "planes — the public one cannot write to the database"},
			},
			Maturity: "The newest of the six and the earliest: designed, not yet running. The architecture, the " +
				"compliance rules and the failure modes are settled; the engine that executes them is being " +
				"built engine-first, so it is delivering feeds to Slack before a single screen exists. What " +
				"makes it more than a plan is that the idea already runs in miniature on this very site, " +
				"which generates /anime.xml and /anime/qotd.xml and posts them on a schedule. AnimeFeedFlux " +
				"is that, generalised.",
		},
	}
}

// FluxProjectBySlug returns the project page for slug, and whether one exists.
func FluxProjectBySlug(slug string) (FluxProject, bool) {
	for _, p := range FluxProjects() {
		if p.Slug == slug {
			return p, true
		}
	}
	return FluxProject{}, false
}

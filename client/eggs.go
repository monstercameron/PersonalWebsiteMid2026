//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/css"
	. "github.com/monstercameron/GoWebComponents/v5/css/u"
	. "github.com/monstercameron/GoWebComponents/v5/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

// The easter eggs are deliberately absent from `help` and from Tab completion: a listed easter egg
// is a menu item, and the reason people screenshot these is that they found them. They are also
// deliberately few — one delightful thing is charming, a dozen turns a portfolio into a theme park.

// runEgg handles the hidden commands. It reports whether it handled the name.
func runEgg(name string, args []string, t *termCtl) bool {
	switch name {
	case "sudo":
		t.emitRows(sudoOut(args))
	case "vim", "vi", "emacs":
		t.emitRows(editorOut(name))
	case "cowsay":
		t.emit(preBlock(cowsay(strings.Join(args, " "))))
	case "fortune":
		t.emitRows([]progRow{row(fortune())})
	case "htop":
		t.emit(preBlock(htop()))
	case "sl":
		startTrain(t)
		return true
	case "matrix", "cmatrix":
		startMatrix(t)
		return true
	case "snake":
		startSnake(t)
		return true
	default:
		return false
	}
	t.emit(gap())
	return true
}

// sudoOut answers `sudo`. The joke has to land without being smug, and it points somewhere real.
func sudoOut(args []string) []progRow {
	if len(args) == 0 {
		return []progRow{{text: "usage: sudo <command>", style: rowErr}}
	}
	return []progRow{
		{text: "cameron is not in the sudoers file. This incident has been reported.", style: rowErr},
		dim("(reported to me. I'm flattered you tried.)"),
	}
}

// editorOut answers the editor commands with the oldest joke in the business.
func editorOut(name string) []progRow {
	if name == "emacs" {
		return []progRow{row("emacs: a great operating system, lacking only a decent editor."),
			dim("no editor here either — use `echo \"text\" > file`.")}
	}
	return []progRow{
		row(name + ": you're in. Good luck getting out."),
		dim("...kidding. :q! accepted, no keystrokes required. Use `echo \"text\" > file` to write."),
	}
}

// cowsay renders the cow. Anything longer than the bubble wraps rather than blowing the box open.
func cowsay(text string) string {
	if text == "" {
		text = "moo"
	}
	const width = 38
	lines := wrapText(text, width)
	longest := 0
	for _, l := range lines {
		if len(l) > longest {
			longest = len(l)
		}
	}
	var b strings.Builder
	b.WriteString(" " + strings.Repeat("_", longest+2) + "\n")
	for i, l := range lines {
		open, close := "|", "|"
		switch {
		case len(lines) == 1:
			open, close = "<", ">"
		case i == 0:
			open, close = "/", "\\"
		case i == len(lines)-1:
			open, close = "\\", "/"
		}
		fmt.Fprintf(&b, "%s %s %s\n", open, padRight(l, longest), close)
	}
	b.WriteString(" " + strings.Repeat("-", longest+2) + "\n")
	b.WriteString("        \\   ^__^\n")
	b.WriteString("         \\  (oo)\\_______\n")
	b.WriteString("            (__)\\       )\\/\\\n")
	b.WriteString("                ||----w |\n")
	b.WriteString("                ||     ||\n")
	return b.String()
}

// wrapText breaks text into lines of at most width characters, on word boundaries.
func wrapText(text string, width int) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur += " " + w
	}
	return append(lines, cur)
}

// fortunes are the `fortune` pool — engineering lines that are actually about how Cam works,
// so even the joke command says something true.
var fortunes = []string{
	"Establish the mechanism first. Everything else is decoration.",
	"The bug is in the layer you were sure about.",
	"A benchmark you did not run is a number you made up.",
	"Weeks of coding can save you hours of planning.",
	"If the fix needs a comment explaining why it works, it does not work yet.",
	"Ship the boring version. Then find out you were wrong about the interesting one.",
	"There are two hard problems: cache invalidation, naming things, and off-by-one errors.",
	"An abstraction with one caller is a function with extra steps.",
}

// fortune returns a random line from the pool.
func fortune() string {
	n := js.Global().Get("Math").Call("random").Float()
	return fortunes[int(n*float64(len(fortunes)))%len(fortunes)]
}

// htop renders a faux process table. The processes named are the ones this page actually runs,
// which makes the joke double as an architecture note.
func htop() string {
	return strings.Join([]string{
		"  PID USER      %CPU %MEM  COMMAND",
		"    1 cameron    0.4  2.1  app.wasm --runtime=go",
		"   14 cameron    0.1  0.6  gwc-reconciler",
		"   27 cameron    0.0  0.3  grpctunnel /socket",
		"   41 cameron    0.0  0.1  vfs (localStorage)",
		"  108 cameron   97.3  8.8  recruiter-attention-span",
		"",
		"press q to quit — or don't, it's a picture",
	}, "\n") + "\n"
}

// preBlock renders a block of pre-formatted text as one scrollback node.
func preBlock(text string) ui.Node {
	return Pre(Class(cFg(), css.Raw("white-space", "pre"), css.Raw("margin", "0"),
		css.Raw("font", "inherit"), css.Raw("line-height", "1.35")),
		strings.TrimRight(text, "\n"))
}

// --- animations ---

// startTrain runs `sl`: the train that punishes you for mistyping `ls`.
func startTrain(t *termCtl) {
	const width = 46
	tok := t.token()
	t.emit(preBlock(""))
	var frame func(x int)
	frame = func(x int) {
		if !t.live(tok) {
			return
		}
		if x < -len(trainArt[0]) {
			t.replaceLast(preBlock(strings.Join(trainArt, "\n")))
			t.emit(gap())
			return
		}
		var b strings.Builder
		for _, line := range trainArt {
			b.WriteString(shiftLine(line, x, width) + "\n")
		}
		t.replaceLast(preBlock(b.String()))
		after(70, func() { frame(x - 2) })
	}
	frame(width)
}

// trainArt is the little engine, one string per row.
var trainArt = []string{
	"      ====        ________",
	"  _D _|  |_______/        \\__I_I_____",
	"   |(_)---  |   H\\________/ |   |    ",
	"   /     |  |   H  |  |     |   |    ",
	"  |      |  |   H  |__--------------|",
	"  | ________|___H__/__|_____/[][]~\\__",
	"  |/ |   |-----------I_____I [][] [] ",
	"__/ =| o |=-~~\\  /~~\\  /~~\\  /~~\\ ____",
	" |/-=|___|=    ||    ||    ||    |_/  ",
	"  \\_/      \\O=====O=====O=====O_/     ",
}

// shiftLine renders one row of art at horizontal offset x within a window of the given width.
func shiftLine(line string, x, width int) string {
	var b strings.Builder
	for col := 0; col < width; col++ {
		i := col - x
		if i >= 0 && i < len(line) {
			b.WriteByte(line[i])
			continue
		}
		b.WriteByte(' ')
	}
	return b.String()
}

// matrixMs is how long the `matrix` rain falls before handing the prompt back. Long enough to be
// the joke, short enough not to be a hostage situation.
const matrixMs = 3200

// startMatrix runs the falling-glyph animation.
func startMatrix(t *termCtl) {
	tok := t.token()
	const rows, cols = 12, 54
	t.emit(preBlock(""))
	elapsed := 0
	var frame func()
	frame = func() {
		if !t.live(tok) {
			return
		}
		if elapsed >= matrixMs {
			t.replaceLast(preBlock("wake up, Neo."))
			t.emit(gap())
			return
		}
		var b strings.Builder
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				b.WriteString(matrixGlyph())
			}
			b.WriteString("\n")
		}
		t.replaceLast(Pre(Class(cGreen(), css.Raw("white-space", "pre"), css.Raw("margin", "0"),
			css.Raw("font", "inherit"), css.Raw("line-height", "1.2")), b.String()))
		elapsed += 90
		after(90, frame)
	}
	frame()
}

// matrixGlyph returns one cell of rain — mostly empty, so the eye reads columns rather than noise.
func matrixGlyph() string {
	n := js.Global().Get("Math").Call("random").Float()
	if n < 0.72 {
		return " "
	}
	const glyphs = "01\uff71\uff72\uff73\uff74\uff75\uff76\uff77\uff78\uff79\uff7a"
	runes := []rune(glyphs)
	return string(runes[int(n*1000)%len(runes)])
}

// --- snake ---

// snakeGame is the one real toy. It installs its own keydown listener rather than routing through
// the terminal's input handling, so the game is entirely self-contained: nothing about the normal
// prompt has to know a game might be running.
type snakeGame struct {
	t          *termCtl
	tok        int
	w, h       int
	body       [][2]int // head first
	dir        [2]int
	food       [2]int
	score      int
	over       bool
	listener   js.Func
	listenerOn bool
}

// startSnake begins a game of snake.
func startSnake(t *termCtl) {
	g := &snakeGame{t: t, tok: t.token(), w: 34, h: 14,
		body: [][2]int{{17, 7}, {16, 7}, {15, 7}}, dir: [2]int{1, 0}}
	g.placeFood()
	t.emitRows([]progRow{head("snake"), dim("arrows or WASD to steer · q to quit")})
	t.emit(preBlock(""))
	g.bindKeys()
	g.tick()
}

// bindKeys installs the game's key listener on the document.
func (g *snakeGame) bindKeys() {
	g.listener = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		// The listener has to check for supersession itself, not rely on tick() to do it: tick
		// only runs every 130ms, so between a new command starting and the next tick unbinding
		// this listener there is a window where a stray q/arrow would still steer or quit a dead
		// game — writing its output into another command's scrollback and swallowing a keystroke
		// (an ArrowUp meant for history recall) on the way.
		if !g.t.live(g.tok) {
			g.unbindKeys()
			return nil
		}
		ev := args[0]
		switch strings.ToLower(ev.Get("key").String()) {
		case "arrowup", "w":
			g.turn(0, -1)
		case "arrowdown", "s":
			g.turn(0, 1)
		case "arrowleft", "a":
			g.turn(-1, 0)
		case "arrowright", "d":
			g.turn(1, 0)
		case "q", "escape":
			g.finish("quit")
			return nil
		default:
			return nil
		}
		// Only swallow the keys the game actually uses, so Ctrl+C and typing still reach the input.
		ev.Call("preventDefault")
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", g.listener)
	g.listenerOn = true
}

// unbindKeys removes the game's key listener. Every exit path goes through here — a game that
// leaves a document listener behind steals the arrow keys from the history recall for good.
func (g *snakeGame) unbindKeys() {
	if !g.listenerOn {
		return
	}
	js.Global().Get("document").Call("removeEventListener", "keydown", g.listener)
	g.listener.Release()
	g.listenerOn = false
}

// turn sets the direction, ignoring a reversal into the snake's own neck.
func (g *snakeGame) turn(dx, dy int) {
	if g.dir[0] == -dx && g.dir[1] == -dy {
		return
	}
	g.dir = [2]int{dx, dy}
}

// placeFood drops food on a random free cell.
func (g *snakeGame) placeFood() {
	rnd := js.Global().Get("Math")
	for i := 0; i < 200; i++ {
		p := [2]int{int(rnd.Call("random").Float() * float64(g.w)), int(rnd.Call("random").Float() * float64(g.h))}
		if !g.occupied(p) {
			g.food = p
			return
		}
	}
}

// occupied reports whether the snake covers a cell.
func (g *snakeGame) occupied(p [2]int) bool {
	for _, b := range g.body {
		if b == p {
			return true
		}
	}
	return false
}

// tick advances the game one step and schedules the next.
func (g *snakeGame) tick() {
	// A new command supersedes the game, same as any other long-running output.
	if !g.t.live(g.tok) {
		g.unbindKeys()
		return
	}
	if g.over {
		return
	}
	headCell := [2]int{g.body[0][0] + g.dir[0], g.body[0][1] + g.dir[1]}
	if headCell[0] < 0 || headCell[0] >= g.w || headCell[1] < 0 || headCell[1] >= g.h || g.occupied(headCell) {
		g.finish("game over")
		return
	}
	g.body = append([][2]int{headCell}, g.body...)
	if headCell == g.food {
		g.score += 10
		g.placeFood()
	} else {
		g.body = g.body[:len(g.body)-1]
	}
	g.render()
	after(130, g.tick)
}

// render draws the board into the last scrollback line.
func (g *snakeGame) render() {
	grid := make([][]byte, g.h)
	for y := range grid {
		grid[y] = []byte(strings.Repeat(" ", g.w))
	}
	grid[g.food[1]][g.food[0]] = '*'
	for i, b := range g.body {
		ch := byte('o')
		if i == 0 {
			ch = '@'
		}
		grid[b[1]][b[0]] = ch
	}
	var sb strings.Builder
	sb.WriteString("+" + strings.Repeat("-", g.w) + "+\n")
	for _, rowBytes := range grid {
		sb.WriteString("|" + string(rowBytes) + "|\n")
	}
	sb.WriteString("+" + strings.Repeat("-", g.w) + "+\n")
	fmt.Fprintf(&sb, "score %d", g.score)
	g.t.replaceLast(preBlock(sb.String()))
}

// finish ends the game, leaving the final board and the score on screen.
func (g *snakeGame) finish(why string) {
	if g.over {
		return
	}
	g.over = true
	g.unbindKeys()
	g.render()
	g.t.emitRows([]progRow{row(why + " — score " + fmt.Sprint(g.score))})
	g.t.emit(gap())
}

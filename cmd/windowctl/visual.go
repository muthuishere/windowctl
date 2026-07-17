// visual.go — CLI entry points for the visual-loop extension verbs:
// find (OCR), scroll, drag, clipboard, and the window-state verbs
// (minimize/maximize/fullscreen/close). Thin flag-parsing shells over
// the public package, same pattern as automation.go.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	windowctl "github.com/muthuishere/windowctl"
)

// findResult is the --json wire shape for one OCR match. Click{X,Y} is
// the point to feed straight into `mouse click` — same global point
// space, no transform (the coordinate contract).
type findResult struct {
	Text       string  `json:"text"`
	Confidence float64 `json:"confidence"`
	ClickX     int     `json:"click_x"`
	ClickY     int     `json:"click_y"`
	X          int     `json:"x"`
	Y          int     `json:"y"`
	Width      int     `json:"w"`
	Height     int     `json:"h"`
}

func findCmd(args []string) {
	fs := flag.NewFlagSet("find", flag.ExitOnError)
	text := fs.String("text", "", "case-insensitive substring to locate on screen (empty = every recognized line)")
	title := fs.String("title", "", "OCR the window matched by title (case-insensitive substring)")
	app := fs.String("app", "", "OCR the window matched by app name (case-insensitive)")
	monitor := fs.Int("monitor", 0, "OCR this monitor (>=1), or base for a relative region")
	x := fs.Int("x", 0, "region x (with --monitor: relative; otherwise: absolute)")
	y := fs.Int("y", 0, "region y (with --monitor: relative; otherwise: absolute)")
	w := fs.Int("w", 0, "region width")
	h := fs.Int("h", 0, "region height")
	asJSON := fs.Bool("json", false, "emit all matches as a JSON array")
	first := fs.Bool("first", false, "print only the top match's click point as \"x y\" (for piping into mouse click)")
	_ = fs.Parse(args)

	if err := rejectEmptyFilterFlags(fs, title, app); err != nil {
		fmt.Fprintln(os.Stderr, "windowctl find:", err)
		os.Exit(2)
	}

	opts := windowctl.FindOptions{Text: *text}
	if *title != "" || *app != "" {
		opts.Match = &windowctl.Match{Title: *title, App: *app}
	}
	regionSet := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "x", "y", "w", "h":
			regionSet = true
		}
	})
	if regionSet {
		if *w <= 0 || *h <= 0 {
			fmt.Fprintln(os.Stderr, "windowctl find: --w and --h must be > 0 for a region")
			os.Exit(2)
		}
		opts.Region = &windowctl.Rect{X: *x, Y: *y, W: *w, H: *h}
	}
	monitorID, err := monitorIDFromFlag(fs, monitor)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl find:", err)
		os.Exit(2)
	}
	opts.Monitor = monitorID

	matches, err := windowctl.FindText(opts)
	if err != nil {
		if errors.Is(err, windowctl.ErrNoMatch) {
			fmt.Fprintln(os.Stderr, "windowctl:", formatNoMatchFilter(*title, *app))
		} else {
			fmt.Fprintln(os.Stderr, "windowctl:", err)
		}
		os.Exit(1)
	}

	if *first {
		// Exit non-zero when nothing matched so scripts can gate on it.
		if len(matches) == 0 {
			fmt.Fprintf(os.Stderr, "windowctl find: no on-screen text matched %q\n", *text)
			os.Exit(1)
		}
		fmt.Printf("%d %d\n", matches[0].ClickX, matches[0].ClickY)
		return
	}
	if *asJSON {
		out := make([]findResult, 0, len(matches))
		for _, m := range matches {
			out = append(out, findResult{
				Text: m.Text, Confidence: m.Confidence,
				ClickX: m.ClickX, ClickY: m.ClickY,
				X: m.Bounds.X, Y: m.Bounds.Y, Width: m.Bounds.W, Height: m.Bounds.H,
			})
		}
		_ = json.NewEncoder(os.Stdout).Encode(out)
		return
	}
	if len(matches) == 0 {
		fmt.Fprintf(os.Stderr, "windowctl find: no on-screen text matched %q\n", *text)
		os.Exit(1)
	}
	for _, m := range matches {
		fmt.Printf("%d\t%d\t%.2f\t%s\n", m.ClickX, m.ClickY, m.Confidence, m.Text)
	}
}

func scrollCmd(args []string) {
	fs := flag.NewFlagSet("scroll", flag.ExitOnError)
	dx := fs.Int("dx", 0, "horizontal wheel lines (positive = right)")
	dy := fs.Int("dy", 0, "vertical wheel lines (positive = content up / wheel away, negative = down)")
	x := fs.Int("x", 0, "cursor x to scroll at (with --monitor: relative; omit x/y to scroll in place)")
	y := fs.Int("y", 0, "cursor y to scroll at (with --monitor: relative; omit x/y to scroll in place)")
	monitor := fs.Int("monitor", 0, "interpret --x/--y relative to this monitor (>=1)")
	_ = fs.Parse(args)

	opts := windowctl.ScrollOptions{DX: *dx, DY: *dy}
	xSet, ySet := false, false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "x":
			xSet = true
		case "y":
			ySet = true
		}
	})
	if xSet != ySet {
		fmt.Fprintln(os.Stderr, "windowctl scroll: --x and --y must be given together")
		os.Exit(2)
	}
	if xSet {
		opts.X, opts.Y = x, y
	}
	monitorID, err := monitorIDFromFlag(fs, monitor)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl scroll:", err)
		os.Exit(2)
	}
	opts.Monitor = monitorID
	if err := windowctl.Scroll(opts); err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
}

func dragCmd(args []string) {
	fs := flag.NewFlagSet("drag", flag.ExitOnError)
	fromX := fs.Int("from-x", 0, "drag start x (with --monitor: relative)")
	fromY := fs.Int("from-y", 0, "drag start y (with --monitor: relative)")
	toX := fs.Int("to-x", 0, "drag end x (with --monitor: relative)")
	toY := fs.Int("to-y", 0, "drag end y (with --monitor: relative)")
	monitor := fs.Int("monitor", 0, "interpret coordinates relative to this monitor (>=1)")
	right := fs.Bool("right", false, "hold the right button")
	middle := fs.Bool("middle", false, "hold the middle button")
	_ = fs.Parse(args)

	need := map[string]bool{"from-x": false, "from-y": false, "to-x": false, "to-y": false}
	fs.Visit(func(f *flag.Flag) {
		if _, ok := need[f.Name]; ok {
			need[f.Name] = true
		}
	})
	for name, set := range need {
		if !set {
			fmt.Fprintf(os.Stderr, "windowctl drag: --%s is required\n", name)
			os.Exit(2)
		}
	}
	if *right && *middle {
		fmt.Fprintln(os.Stderr, "windowctl drag: --right and --middle are mutually exclusive")
		os.Exit(2)
	}
	monitorID, err := monitorIDFromFlag(fs, monitor)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl drag:", err)
		os.Exit(2)
	}
	opts := windowctl.DragOptions{
		Monitor: monitorID,
		FromX:   *fromX, FromY: *fromY, ToX: *toX, ToY: *toY,
	}
	if *right {
		opts.Button = windowctl.MouseRight
	}
	if *middle {
		opts.Button = windowctl.MouseMiddle
	}
	if err := windowctl.Drag(opts); err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
}

func clipboardCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: windowctl clipboard (get | set --text <s> | set (reads stdin))")
		os.Exit(2)
	}
	switch args[0] {
	case "get":
		s, err := windowctl.GetClipboard()
		if err != nil {
			fmt.Fprintln(os.Stderr, "windowctl:", err)
			os.Exit(1)
		}
		// No trailing newline: the clipboard's bytes verbatim.
		fmt.Print(s)
	case "set":
		fs := flag.NewFlagSet("clipboard set", flag.ExitOnError)
		text := fs.String("text", "", "text to place on the clipboard (omit to read from stdin)")
		_ = fs.Parse(args[1:])
		val := *text
		textPassed := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "text" {
				textPassed = true
			}
		})
		if !textPassed {
			b, err := io.ReadAll(os.Stdin)
			if err != nil {
				fmt.Fprintln(os.Stderr, "windowctl clipboard set:", err)
				os.Exit(1)
			}
			val = string(b)
		}
		if err := windowctl.SetClipboard(val); err != nil {
			fmt.Fprintln(os.Stderr, "windowctl:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "windowctl clipboard: unknown subcommand %q (want get or set)\n", args[0])
		os.Exit(2)
	}
}

// windowStateCmd handles the four state verbs, dispatched by verb name
// (minimize/maximize/fullscreen/close) — each requires a --title/--app
// filter to pick the window.
func windowStateCmd(verb string, args []string) {
	fs := flag.NewFlagSet(verb, flag.ExitOnError)
	title := fs.String("title", "", "target window by title (case-insensitive substring)")
	app := fs.String("app", "", "target window by app name (case-insensitive)")
	_ = fs.Parse(args)

	if err := rejectEmptyFilterFlags(fs, title, app); err != nil {
		fmt.Fprintf(os.Stderr, "windowctl %s: %v\n", verb, err)
		os.Exit(2)
	}
	if *title == "" && *app == "" {
		fmt.Fprintf(os.Stderr, "windowctl %s: --title or --app is required\n", verb)
		os.Exit(2)
	}
	op, err := windowctl.ParseWindowOp(verb)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(2)
	}
	if err := windowctl.SetWindowState(windowctl.Match{Title: *title, App: *app}, op); err != nil {
		if errors.Is(err, windowctl.ErrNoMatch) {
			fmt.Fprintln(os.Stderr, "windowctl:", formatNoMatchFilter(*title, *app))
		} else {
			fmt.Fprintln(os.Stderr, "windowctl:", err)
		}
		os.Exit(1)
	}
}

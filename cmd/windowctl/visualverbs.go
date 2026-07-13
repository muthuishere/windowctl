// visualverbs.go — CLI entry points for the COMPOSITE, text-targeted
// verbs: click (OCR → click), exists (OCR → gate), read (OCR → dump).
// Each is one self-contained call that screenshots, OCRs, resolves the
// target, and acts — so an agent drives the GUI by the words on screen
// and never hand-computes a coordinate. Thin flag-parsing shells over
// the public package, same pattern as automation.go / visual.go.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	windowctl "github.com/muthuishere/windowctl"
)

// scopeFlags is the shared "where to OCR" flag set that click/exists/read
// all accept: a window filter (--title/--app), or a monitor, or a
// region (relative to --monitor when set, else absolute) — mirroring
// screenshot/find exactly so the coordinate contract is identical.
type scopeFlags struct {
	title, app *string
	monitor    *int
	x, y, w, h *int
}

func addScopeFlags(fs *flag.FlagSet) scopeFlags {
	return scopeFlags{
		title:   fs.String("title", "", "scope OCR to the window matched by title (case-insensitive substring)"),
		app:     fs.String("app", "", "scope OCR to the window matched by app name (case-insensitive)"),
		monitor: fs.Int("monitor", 0, "scope OCR to this monitor (>=1), or base for a relative region"),
		x:       fs.Int("x", 0, "region x (with --monitor: relative; otherwise: absolute)"),
		y:       fs.Int("y", 0, "region y (with --monitor: relative; otherwise: absolute)"),
		w:       fs.Int("w", 0, "region width"),
		h:       fs.Int("h", 0, "region height"),
	}
}

// resolve builds a windowctl.FindOptions from the parsed scope flags,
// applying the same validation the find/screenshot commands use.
func (s scopeFlags) resolve(fs *flag.FlagSet, text string) (windowctl.FindOptions, error) {
	if err := rejectEmptyFilterFlags(fs, s.title, s.app); err != nil {
		return windowctl.FindOptions{}, err
	}
	opts := windowctl.FindOptions{Text: text}
	if *s.title != "" || *s.app != "" {
		opts.Match = &windowctl.Match{Title: *s.title, App: *s.app}
	}
	regionSet := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "x", "y", "w", "h":
			regionSet = true
		}
	})
	if regionSet {
		if *s.w <= 0 || *s.h <= 0 {
			return windowctl.FindOptions{}, fmt.Errorf("--w and --h must be > 0 for a region")
		}
		opts.Region = &windowctl.Rect{X: *s.x, Y: *s.y, W: *s.w, H: *s.h}
	}
	monitorID, err := monitorIDFromFlag(fs, s.monitor)
	if err != nil {
		return windowctl.FindOptions{}, err
	}
	opts.Monitor = monitorID
	return opts, nil
}

func clickCmd(args []string) {
	fs := flag.NewFlagSet("click", flag.ExitOnError)
	text := fs.String("text", "", "on-screen text label to click (case-insensitive substring)")
	scope := addScopeFlags(fs)
	right := fs.Bool("right", false, "right-click")
	middle := fs.Bool("middle", false, "middle-click")
	double := fs.Bool("double", false, "double-click")
	asJSON := fs.Bool("json", false, "emit the clicked match as JSON")
	_ = fs.Parse(args)

	if *text == "" {
		fmt.Fprintln(os.Stderr, "windowctl click: --text is required")
		os.Exit(2)
	}
	if *right && *middle {
		fmt.Fprintln(os.Stderr, "windowctl click: --right and --middle are mutually exclusive")
		os.Exit(2)
	}
	find, err := scope.resolve(fs, *text)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl click:", err)
		os.Exit(2)
	}
	opts := windowctl.ClickTextOptions{
		Text:    find.Text,
		Monitor: find.Monitor,
		Region:  find.Region,
		Match:   find.Match,
		Double:  *double,
	}
	if *right {
		opts.Button = windowctl.MouseRight
	}
	if *middle {
		opts.Button = windowctl.MouseMiddle
	}
	m, err := windowctl.ClickText(opts)
	if err != nil {
		// ClickText's ErrNoMatch already carries the searched text; a
		// window-filter ErrNoMatch would too. Surface verbatim either way.
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
	if *asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(findResult{
			Text: m.Text, Confidence: m.Confidence,
			ClickX: m.ClickX, ClickY: m.ClickY,
			X: m.Bounds.X, Y: m.Bounds.Y, Width: m.Bounds.W, Height: m.Bounds.H,
		})
		return
	}
	fmt.Printf("clicked %q at %d,%d (confidence %.2f)\n", m.Text, m.ClickX, m.ClickY, m.Confidence)
}

func existsCmd(args []string) {
	fs := flag.NewFlagSet("exists", flag.ExitOnError)
	text := fs.String("text", "", "on-screen text to look for (case-insensitive substring)")
	scope := addScopeFlags(fs)
	asJSON := fs.Bool("json", false, "emit {\"found\":bool,...} as JSON")
	_ = fs.Parse(args)

	if *text == "" {
		fmt.Fprintln(os.Stderr, "windowctl exists: --text is required")
		os.Exit(2)
	}
	find, err := scope.resolve(fs, *text)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl exists:", err)
		os.Exit(2)
	}
	found, m, err := windowctl.TextExists(find)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
	if *asJSON {
		out := map[string]any{"found": found}
		if found {
			out["text"] = m.Text
			out["click_x"] = m.ClickX
			out["click_y"] = m.ClickY
			out["confidence"] = m.Confidence
		}
		_ = json.NewEncoder(os.Stdout).Encode(out)
	} else if found {
		fmt.Printf("found %q at %d,%d\n", m.Text, m.ClickX, m.ClickY)
	} else {
		fmt.Printf("not found: %q\n", *text)
	}
	// Script-friendly exit code: 0 = present, 1 = absent.
	if !found {
		os.Exit(1)
	}
}

func readCmd(args []string) {
	fs := flag.NewFlagSet("read", flag.ExitOnError)
	text := fs.String("text", "", "only lines containing this substring (omit = every recognized line)")
	scope := addScopeFlags(fs)
	asJSON := fs.Bool("json", false, "emit all lines as a JSON array")
	_ = fs.Parse(args)

	find, err := scope.resolve(fs, *text)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl read:", err)
		os.Exit(2)
	}
	lines, err := windowctl.ReadScreen(find)
	if err != nil {
		if errors.Is(err, windowctl.ErrNoMatch) {
			fmt.Fprintln(os.Stderr, "windowctl:", formatNoMatchFilter(*scope.title, *scope.app))
		} else {
			fmt.Fprintln(os.Stderr, "windowctl:", err)
		}
		os.Exit(1)
	}
	if *asJSON {
		out := make([]findResult, 0, len(lines))
		for _, m := range lines {
			out = append(out, findResult{
				Text: m.Text, Confidence: m.Confidence,
				ClickX: m.ClickX, ClickY: m.ClickY,
				X: m.Bounds.X, Y: m.Bounds.Y, Width: m.Bounds.W, Height: m.Bounds.H,
			})
		}
		_ = json.NewEncoder(os.Stdout).Encode(out)
		return
	}
	for _, m := range lines {
		fmt.Println(m.Text)
	}
}

package windowctl

import (
	"fmt"
	"sort"
)

// This file holds the COMPOSITE, text-targeted verbs — the ones an agent
// reaches for so it never hand-computes a coordinate. Each composes the
// already-tested primitives (FindText → MouseClick, FindText alone) and
// lives here, in the OS-agnostic public layer, not in any adapter.
//
// The load-bearing invariant is COORDINATE UNIFICATION. There are two
// coordinate spaces in this codebase and mixing them is the classic bug:
//
//   - Window.Bounds and the OCR capture rect are ABSOLUTE global points
//     (virtual-desktop space; the same space monitors report their
//     origins in).
//   - `--monitor N` on mouse/move verbs switches raw x/y to be RELATIVE
//     to monitor N's origin.
//
// FindText already returns each match's ClickX/ClickY in ABSOLUTE global
// points (the adapter adds the capture rect's origin before returning).
// So the composite verbs feed those straight to MouseClick with NO
// monitor set — which MouseClick also interprets as absolute global. The
// whole path stays in ONE space end to end; the caller never sees, and
// never has to reconcile, the relative-vs-global distinction.

// ClickTextOptions targets an on-screen text label to click. The scope
// fields (Monitor/Region/Match) are identical to FindOptions — same
// resolution and mutual-exclusion rules — and select WHERE to OCR.
// Text is the label to click (case-insensitive substring); the
// highest-confidence match's click point is used.
type ClickTextOptions struct {
	Text    string
	Monitor *int
	Region  *Rect
	Match   *Match
	Button  MouseButton
	Double  bool
}

// ClickText OCRs the resolved scope, picks the highest-confidence match
// for opts.Text, and clicks its click point. It returns the TextMatch
// that was clicked so callers can log/verify what was hit.
//
// When opts.Match names a window, that window is raised first so the OCR
// reads the intended window (and the click lands in it) rather than
// whatever happens to be stacked on top.
//
// The click point comes straight from FindText in absolute global points
// and is handed to MouseClick with no monitor — one coordinate space,
// no transform (see the file header). Returns an error wrapping
// ErrNoMatch when no on-screen text matches opts.Text (distinct from a
// window-filter ErrNoMatch), ErrScreenCaptureDenied without the macOS
// Screen Recording grant, ErrNotImplemented where OCR is unavailable.
func ClickText(opts ClickTextOptions) (TextMatch, error) {
	return clickTextWith(defaultAdapter, opts)
}

func clickTextWith(a Adapter, opts ClickTextOptions) (TextMatch, error) {
	if opts.Text == "" {
		return TextMatch{}, fmt.Errorf("click: text is required (nothing to locate)")
	}
	// Raise the target window first so OCR reads it (not an occluding
	// window) and the click lands in the intended app.
	if opts.Match != nil {
		if err := focusWith(a, *opts.Match); err != nil {
			return TextMatch{}, err
		}
	}
	matches, err := findTextWith(a, FindOptions{
		Text:    opts.Text,
		Monitor: opts.Monitor,
		Region:  opts.Region,
		Match:   opts.Match,
	})
	if err != nil {
		return TextMatch{}, err
	}
	if len(matches) == 0 {
		return TextMatch{}, fmt.Errorf("%w: no on-screen text matched %q", ErrNoMatch, opts.Text)
	}
	top := matches[0]
	if err := mouseClickWith(a, ClickOptions{
		X:      &top.ClickX,
		Y:      &top.ClickY,
		Button: opts.Button,
		Double: opts.Double,
	}); err != nil {
		return TextMatch{}, err
	}
	return top, nil
}

// TextExists reports whether opts.Text appears anywhere in the resolved
// OCR scope, returning the top match when it does. It is ClickText's
// read-only sibling (identical scoping) — the primitive for gating a
// step ("is the Save dialog up yet?") without clicking anything.
func TextExists(opts FindOptions) (bool, TextMatch, error) {
	return textExistsWith(defaultAdapter, opts)
}

func textExistsWith(a Adapter, opts FindOptions) (bool, TextMatch, error) {
	if opts.Text == "" {
		return false, TextMatch{}, fmt.Errorf("exists: text is required")
	}
	matches, err := findTextWith(a, opts)
	if err != nil {
		return false, TextMatch{}, err
	}
	if len(matches) == 0 {
		return false, TextMatch{}, nil
	}
	return true, matches[0], nil
}

// readingRowBand is the vertical tolerance (in points) within which two
// text runs are treated as the same row for reading-order sorting, so a
// label a few pixels higher on the right doesn't jump ahead of one on
// the left.
const readingRowBand = 12

// ReadScreen OCRs the resolved scope and returns every recognized run in
// READING ORDER (top-to-bottom, then left-to-right within a row) — the
// "dump what this window says" primitive behind `windowctl read`. Unlike
// FindText (which sorts by confidence for click-targeting), reading
// order is what an agent wants when consuming the text as a document.
// An empty opts.Text reads everything (the usual case); a non-empty
// Text filters to matching runs first.
func ReadScreen(opts FindOptions) ([]TextMatch, error) {
	return readScreenWith(defaultAdapter, opts)
}

func readScreenWith(a Adapter, opts FindOptions) ([]TextMatch, error) {
	matches, err := findTextWith(a, opts)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(matches, func(i, j int) bool {
		bi, bj := matches[i].Bounds, matches[j].Bounds
		if absInt(bi.Y-bj.Y) > readingRowBand {
			return bi.Y < bj.Y
		}
		return bi.X < bj.X
	})
	return matches, nil
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

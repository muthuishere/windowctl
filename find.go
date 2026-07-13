package windowctl

import (
	"sort"
	"strings"
)

// FindOptions selects WHERE FindText looks and WHAT it matches. The
// scope fields mirror ScreenshotOptions exactly (same resolution rules,
// same mutual-exclusion): Match a window, an explicit Region (monitor-
// relative when Monitor is also set), a Monitor alone, or — with none
// set — the focused monitor. Text is a case-insensitive substring; empty
// Text returns every recognized line (useful for "what's on screen?").
type FindOptions struct {
	Text    string
	Monitor *int
	Region  *Rect
	Match   *Match
}

// FindText runs native on-screen OCR over the resolved scope and returns
// the matching text runs, sorted by confidence (highest first). Every
// TextMatch carries Bounds and Click in absolute global points — the
// SAME space MouseClick consumes — so the visual loop is literally
// `find --text "Submit"` → take the top match's Click → `mouse click`,
// with no pixel math and no scale factor even on Retina displays (the
// capture is point-normalized, 1 image pixel == 1 global point).
//
// Returns ErrScreenCaptureDenied without the macOS Screen Recording
// grant, ErrNotImplemented on platforms without a native OCR path.
func FindText(opts FindOptions) ([]TextMatch, error) {
	return findTextWith(defaultAdapter, opts)
}

func findTextWith(a Adapter, opts FindOptions) ([]TextMatch, error) {
	// Reuse the screenshot scope resolver so `find` and `screenshot`
	// agree on exactly what "monitor 2" / "this window" / "this region"
	// mean — the coordinate contract is shared, not reimplemented.
	rect, err := resolveScreenshotRect(a, ScreenshotOptions{
		Monitor: opts.Monitor,
		Region:  opts.Region,
		Match:   opts.Match,
	})
	if err != nil {
		return nil, err
	}
	all, err := a.FindText(rect)
	if err != nil {
		return nil, err
	}
	q := strings.ToLower(strings.TrimSpace(opts.Text))
	out := make([]TextMatch, 0, len(all))
	for _, m := range all {
		if q == "" || strings.Contains(strings.ToLower(m.Text), q) {
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Confidence > out[j].Confidence
	})
	return out, nil
}

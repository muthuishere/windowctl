package windowctl

import (
	"errors"
	"testing"
	"time"
)

// ClickText feeds FindText's absolute-global click point STRAIGHT to
// MouseClick with no monitor — proving the coordinate-unification
// contract: the click lands at exactly the match's global point, with no
// monitor-relative offset ever applied. This is the whole reason an agent
// can `click --text` without touching a coordinate.
func TestClickTextClicksTopMatchInGlobalSpace(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = []TextMatch{
		{Text: "Cancel", Confidence: 0.70, ClickX: 100, ClickY: 100},
		// Highest confidence, and its click point sits on monitor 2
		// (x >= 1920) in GLOBAL space.
		{Text: "Submit", Confidence: 0.95, ClickX: 2400, ClickY: 900},
	}
	m, err := clickTextWith(a, ClickTextOptions{Text: "submit"})
	if err != nil {
		t.Fatal(err)
	}
	if m.Text != "Submit" {
		t.Fatalf("clicked the wrong match: %+v", m)
	}
	if a.clickedAt == nil || *a.clickedAt != [2]int{2400, 900} {
		t.Fatalf("clicked at %v, want the match's GLOBAL point [2400 900] with no offset", a.clickedAt)
	}
}

// When scoped to a window, ClickText raises that window BEFORE OCR (so it
// reads the intended window, not an occluding one) and OCRs the window's
// bounds.
func TestClickTextRaisesWindowScopeFirst(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = []TextMatch{{Text: "OK", Confidence: 0.9, ClickX: 300, ClickY: 300}}
	if _, err := clickTextWith(a, ClickTextOptions{Text: "ok", Match: &Match{App: "google chrome"}}); err != nil {
		t.Fatal(err)
	}
	if a.focused != "w1" {
		t.Fatalf("window scope must be raised before OCR; Focus not called (focused=%q)", a.focused)
	}
	if want := (Rect{X: 100, Y: 50, W: 800, H: 600}); a.findRect != want {
		t.Fatalf("OCR rect %+v, want the matched window's bounds %+v", a.findRect, want)
	}
	if a.clickedAt == nil || *a.clickedAt != [2]int{300, 300} {
		t.Fatalf("clicked at %v, want [300 300]", a.clickedAt)
	}
}

// No on-screen text matching the query is a hard, ErrNoMatch-wrapped
// error — and nothing is clicked.
func TestClickTextNoMatchErrorsAndClicksNothing(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = []TextMatch{{Text: "Nope", Confidence: 0.9}}
	_, err := clickTextWith(a, ClickTextOptions{Text: "absent"})
	if !errors.Is(err, ErrNoMatch) {
		t.Fatalf("want ErrNoMatch, got %v", err)
	}
	if a.clickedAt != nil {
		t.Fatalf("nothing should be clicked on a miss, clicked at %v", a.clickedAt)
	}
}

func TestClickTextRequiresText(t *testing.T) {
	a := newAutomationAdapter()
	if _, err := clickTextWith(a, ClickTextOptions{}); err == nil {
		t.Fatal("want error for empty text")
	}
}

// TextExists is the read-only gate: true + the top match when present,
// false when absent, never clicking.
func TestTextExists(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = []TextMatch{{Text: "Document Saved", Confidence: 0.9, ClickX: 5, ClickY: 6}}

	found, m, err := textExistsWith(a, FindOptions{Text: "saved"})
	if err != nil {
		t.Fatal(err)
	}
	if !found || m.Text != "Document Saved" {
		t.Fatalf("want found match, got found=%v m=%+v", found, m)
	}
	found, _, err = textExistsWith(a, FindOptions{Text: "unsaved changes"})
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("want not found for absent text")
	}
	if a.clickedAt != nil {
		t.Fatal("exists must never click")
	}
}

func TestTextExistsRequiresText(t *testing.T) {
	a := newAutomationAdapter()
	if _, _, err := textExistsWith(a, FindOptions{}); err == nil {
		t.Fatal("want error for empty text")
	}
}

// ReadScreen returns runs in READING ORDER (top-to-bottom, then
// left-to-right within a row band) — NOT FindText's confidence order.
func TestReadScreenReadingOrder(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = []TextMatch{
		{Text: "bottom-left", Confidence: 0.99, Bounds: Rect{X: 10, Y: 500}},
		// Same row as top-left (|10-12| <= readingRowBand) but further right.
		{Text: "top-right", Confidence: 0.50, Bounds: Rect{X: 800, Y: 10}},
		{Text: "top-left", Confidence: 0.10, Bounds: Rect{X: 10, Y: 12}},
	}
	got, err := readScreenWith(a, FindOptions{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"top-left", "top-right", "bottom-left"}
	if len(got) != len(want) {
		t.Fatalf("got %d runs, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Text != want[i] {
			t.Fatalf("reading order %d = %q, want %q (full: %+v)", i, got[i].Text, want[i], got)
		}
	}
}

// TestTypeIntoVerifiesFocusBeforeTyping is the positive counterpart to
// TestTypeIntoRefusesWhenFocusNeverLands: the target window is NOT yet
// focused, so the guard must RAISE it, confirm focus actually landed via
// the window list, and only THEN type. The mock models a real focus
// transfer, so we can assert that at the instant the first keystroke
// fired, focus had already moved to the target's monitor — i.e. focus is
// verified BEFORE typing, never after.
func TestTypeIntoVerifiesFocusBeforeTyping(t *testing.T) {
	a := newAutomationAdapter()
	// w1 is on monitor 1; monitor 2 currently holds focus → not focused yet.
	a.focusSetsMonitorFocused = true

	oldSleep, oldNow, oldInterval := sleepFn, nowFn, waitPollInterval
	waitPollInterval = 0
	sleepFn = func(time.Duration) {}
	defer func() { sleepFn, nowFn, waitPollInterval = oldSleep, oldNow, oldInterval }()

	if err := typeIntoWith(a, Match{App: "google chrome"}, "hello"); err != nil {
		t.Fatal(err)
	}
	if a.focused != "w1" {
		t.Fatalf("an unfocused target must be raised before typing; Focus not called")
	}
	if a.typedText != "hello" {
		t.Fatalf("text should type once focus is verified, got %q", a.typedText)
	}
	// Monitor 1 holds w1's centroid; focus must have transferred there
	// BEFORE the keystroke landed.
	if a.typedFocusedMonitor != 1 {
		t.Fatalf("focus not verified before typing: keystroke fired with monitor %d focused, want 1", a.typedFocusedMonitor)
	}
}

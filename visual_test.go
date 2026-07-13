package windowctl

import (
	"testing"
	"time"
)

// ensureFocused settles for the configured duration AFTER the window
// reports focused, so the first keystrokes don't race the responder
// change. WCTL_FOCUS_SETTLE_MS overrides the default.
func TestFocusSettleAppliedAfterFocusConfirmed(t *testing.T) {
	a := newAutomationAdapter()
	a.monitors[0].Focused = true
	a.monitors[1].Focused = false

	oldSleep := sleepFn
	var slept []time.Duration
	sleepFn = func(d time.Duration) { slept = append(slept, d) }
	defer func() { sleepFn = oldSleep }()

	t.Setenv("WCTL_FOCUS_SETTLE_MS", "42")
	if err := ensureFocused(a, Match{App: "google chrome"}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range slept {
		if d == 42*time.Millisecond {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a 42ms settle sleep, got %v", slept)
	}
}

func TestFocusSettleDurationParsing(t *testing.T) {
	t.Setenv("WCTL_FOCUS_SETTLE_MS", "0")
	if got := focusSettleDuration(); got != 0 {
		t.Fatalf("0 should disable settle, got %v", got)
	}
	t.Setenv("WCTL_FOCUS_SETTLE_MS", "garbage")
	if got := focusSettleDuration(); got != defaultFocusSettleMS*time.Millisecond {
		t.Fatalf("malformed value should fall back to default, got %v", got)
	}
}

// FindText resolves its scope through the same resolver as Screenshot,
// filters by case-insensitive substring, and sorts by confidence.
func TestFindTextFiltersAndSortsByConfidence(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = []TextMatch{
		{Text: "Cancel", Confidence: 0.99, ClickX: 10, ClickY: 20},
		{Text: "Submit form", Confidence: 0.80, ClickX: 30, ClickY: 40},
		{Text: "submit", Confidence: 0.95, ClickX: 50, ClickY: 60},
	}
	got, err := findTextWith(a, FindOptions{Text: "SUBMIT"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 matches for substring 'submit', got %d: %+v", len(got), got)
	}
	// Highest confidence first: "submit" (0.95) before "Submit form" (0.80).
	if got[0].Text != "submit" || got[1].Text != "Submit form" {
		t.Fatalf("wrong sort order: %+v", got)
	}
}

// Empty Text returns every recognized line (a "what's on screen" read).
func TestFindTextEmptyQueryReturnsAll(t *testing.T) {
	a := newAutomationAdapter()
	a.findMatches = []TextMatch{{Text: "A", Confidence: 0.5}, {Text: "B", Confidence: 0.6}}
	got, err := findTextWith(a, FindOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want all 2 lines, got %d", len(got))
	}
}

// The scope resolver defaults to the focused monitor, so FindText OCRs
// exactly the rect Screenshot would capture — the shared coordinate
// contract that lets a found Click feed straight into MouseClick.
func TestFindTextDefaultsToFocusedMonitorRect(t *testing.T) {
	a := newAutomationAdapter()
	if _, err := findTextWith(a, FindOptions{Text: "x"}); err != nil {
		t.Fatal(err)
	}
	if want := (Rect{X: 1920, Y: 0, W: 2560, H: 1440}); a.findRect != want {
		t.Fatalf("OCR rect %+v, want focused-monitor rect %+v", a.findRect, want)
	}
}

// Scroll at an explicit monitor-relative point offsets by the monitor
// origin (same rule as MouseClick / MoveCoords).
func TestScrollMonitorRelativePoint(t *testing.T) {
	a := newAutomationAdapter()
	id := 2
	x, y := 100, 200
	if err := scrollWith(a, ScrollOptions{Monitor: &id, X: &x, Y: &y, DY: 3}); err != nil {
		t.Fatal(err)
	}
	if a.scrolledAt == nil || *a.scrolledAt != [2]int{2020, 200} {
		t.Fatalf("scrolled at %v, want [2020 200] (monitor-2 origin + point)", a.scrolledAt)
	}
	if *a.scrollDelta != [2]int{0, 3} {
		t.Fatalf("scroll delta %v, want [0 3]", a.scrollDelta)
	}
}

// Scroll with no point uses the current cursor position.
func TestScrollNoPointUsesCursor(t *testing.T) {
	a := newAutomationAdapter()
	a.cursorX, a.cursorY = 777, 888
	if err := scrollWith(a, ScrollOptions{DY: -2}); err != nil {
		t.Fatal(err)
	}
	if *a.scrolledAt != [2]int{777, 888} {
		t.Fatalf("scrolled at %v, want cursor [777 888]", a.scrolledAt)
	}
}

func TestScrollRejectsZeroDelta(t *testing.T) {
	a := newAutomationAdapter()
	if err := scrollWith(a, ScrollOptions{}); err == nil {
		t.Fatal("want error for zero dx+dy")
	}
}

// Drag resolves both endpoints with the monitor-relative rule.
func TestDragMonitorRelativeEndpoints(t *testing.T) {
	a := newAutomationAdapter()
	id := 2
	err := dragWith(a, DragOptions{Monitor: &id, FromX: 10, FromY: 20, ToX: 30, ToY: 40, Button: MouseLeft})
	if err != nil {
		t.Fatal(err)
	}
	if *a.draggedFrom != [2]int{1930, 20} || *a.draggedTo != [2]int{1950, 40} {
		t.Fatalf("drag %v -> %v, want [1930 20] -> [1950 40]", a.draggedFrom, a.draggedTo)
	}
}

// Maximize is composed as a Move to the window's monitor bounds and must
// never reach the adapter's WindowState op.
func TestSetWindowStateMaximizeComposesMove(t *testing.T) {
	a := newAutomationAdapter()
	// Window on monitor 1 (bounds 100,50 → centroid on monitor 1).
	if err := setWindowStateWith(a, Match{App: "Google Chrome"}, WindowMaximize); err != nil {
		t.Fatal(err)
	}
	if a.windowStateOp != nil {
		t.Fatalf("maximize should not call adapter WindowState, got op %v", *a.windowStateOp)
	}
	if got := a.moved["w1"]; got != (Rect{X: 0, Y: 0, W: 1920, H: 1080}) {
		t.Fatalf("maximize moved to %+v, want full monitor-1 bounds", got)
	}
}

// Minimize/fullscreen/close route to the adapter WindowState with the
// matched window's ID.
func TestSetWindowStateRoutesToAdapter(t *testing.T) {
	for _, op := range []WindowOp{WindowMinimize, WindowFullscreen, WindowClose} {
		a := newAutomationAdapter()
		if err := setWindowStateWith(a, Match{App: "Google Chrome"}, op); err != nil {
			t.Fatal(err)
		}
		if a.windowStateID != "w1" || a.windowStateOp == nil || *a.windowStateOp != op {
			t.Fatalf("op %v: routed id=%q op=%v", op, a.windowStateID, a.windowStateOp)
		}
	}
}

func TestParseWindowOp(t *testing.T) {
	cases := map[string]WindowOp{
		"minimize":   WindowMinimize,
		"maximize":   WindowMaximize,
		"fullscreen": WindowFullscreen,
		"close":      WindowClose,
	}
	for verb, want := range cases {
		got, err := ParseWindowOp(verb)
		if err != nil || got != want {
			t.Fatalf("ParseWindowOp(%q) = %v, %v; want %v", verb, got, err, want)
		}
	}
	if _, err := ParseWindowOp("wiggle"); err == nil {
		t.Fatal("want error for unknown verb")
	}
}

// Clipboard round-trips through the adapter.
func TestClipboardRoundTrip(t *testing.T) {
	a := newAutomationAdapter()
	if err := a.SetClipboard("hello"); err != nil {
		t.Fatal(err)
	}
	got, err := a.Clipboard()
	if err != nil {
		t.Fatal(err)
	}
	if got != "hello" {
		t.Fatalf("clipboard = %q, want hello", got)
	}
}

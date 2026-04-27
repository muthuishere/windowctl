package windowctl

import (
	"errors"
	"strings"
	"testing"

	"github.com/muthuishere/windowctl/internal/core"
)

type mockAdapter struct {
	windows           []Window
	monitors          []Monitor
	moved             map[string]Rect
	focused           string
	listErr           error
	moveErr           error
	focusErr          error
	requestAXErr      error
	requestAXCalled   int
}

func (m *mockAdapter) ListWindows() ([]Window, error)  { return m.windows, m.listErr }
func (m *mockAdapter) ListMonitors() ([]Monitor, error) { return m.monitors, nil }
func (m *mockAdapter) Move(id string, b Rect) error {
	if m.moveErr != nil {
		return m.moveErr
	}
	if m.moved == nil {
		m.moved = map[string]Rect{}
	}
	m.moved[id] = b
	return nil
}
func (m *mockAdapter) Focus(id string) error {
	if m.focusErr != nil {
		return m.focusErr
	}
	m.focused = id
	return nil
}
func (m *mockAdapter) RequestAccessibility() error {
	m.requestAXCalled++
	return m.requestAXErr
}

func sampleWindows() []Window {
	return []Window{
		{ID: "1", Title: "Inbox - Chrome", App: "Google Chrome"},
		{ID: "2", Title: "main.go - VSCode", App: "Code"},
		{ID: "3", Title: "Slack | general", App: "Slack"},
	}
}

func TestApplyFilterByTitleIsPartialAndCaseInsensitive(t *testing.T) {
	out := applyFilter(sampleWindows(), Filter{Title: "ChRoMe"})
	if len(out) != 1 || out[0].ID != "1" {
		t.Fatalf("expected the chrome window, got %+v", out)
	}
}

func TestApplyFilterByAppIsCaseInsensitiveExactMatch(t *testing.T) {
	out := applyFilter(sampleWindows(), Filter{App: "slack"})
	if len(out) != 1 || out[0].ID != "3" {
		t.Fatalf("expected the slack window by exact app match, got %+v", out)
	}

	out = applyFilter(sampleWindows(), Filter{App: "Sla"})
	if len(out) != 0 {
		t.Fatalf("expected no match for app prefix (exact match required), got %+v", out)
	}
}

func TestApplyFilterCombinesTitleAndApp(t *testing.T) {
	ws := append(sampleWindows(), Window{ID: "4", Title: "Dashboards", App: "Google Chrome"})
	out := applyFilter(ws, Filter{Title: "inbox", App: "Google Chrome"})
	if len(out) != 1 || out[0].ID != "1" {
		t.Fatalf("expected only the chrome+inbox window, got %+v", out)
	}
}

func TestListWindowsReturnsAllWhenNoFilter(t *testing.T) {
	a := &mockAdapter{windows: sampleWindows()}
	out, err := listWindowsWith(a, Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Fatalf("expected 3 windows, got %d", len(out))
	}
}

func TestMoveDelegatesToAdapter(t *testing.T) {
	a := &mockAdapter{windows: sampleWindows()}
	if err := moveWith(a, Match{Title: "chrome"}, Target{Bounds: Rect{X: 10, Y: 20, W: 100, H: 200}}); err != nil {
		t.Fatal(err)
	}
	if got := a.moved["1"]; got != (Rect{X: 10, Y: 20, W: 100, H: 200}) {
		t.Fatalf("unexpected move bounds: %+v", got)
	}
}

func TestMoveReturnsErrNoMatchWhenFilterMatchesNothing(t *testing.T) {
	a := &mockAdapter{windows: sampleWindows()}
	err := moveWith(a, Match{Title: "nope"}, Target{Bounds: Rect{}})
	if !errors.Is(err, ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

func TestFocusDelegatesToAdapter(t *testing.T) {
	a := &mockAdapter{windows: sampleWindows()}
	if err := focusWith(a, Match{App: "Code"}); err != nil {
		t.Fatal(err)
	}
	if a.focused != "2" {
		t.Fatalf("expected window 2 to be focused, got %q", a.focused)
	}
}

func TestFocusReturnsErrNoMatchWhenFilterMatchesNothing(t *testing.T) {
	a := &mockAdapter{windows: sampleWindows()}
	err := focusWith(a, Match{Title: "nope"})
	if !errors.Is(err, ErrNoMatch) {
		t.Fatalf("expected ErrNoMatch, got %v", err)
	}
}

// TestErrAccessibilityDeniedSentinelExists guards the contract that the
// macOS AX-denial path returns a typed sentinel rather than an ad-hoc
// error string. Required by docs/specs/01-os-spikes.md "Accessibility
// permission denied on macOS" and requirements.md §11 (the row that
// mandates instructing the user to grant Accessibility access).
func TestErrAccessibilityDeniedSentinelExists(t *testing.T) {
	if ErrAccessibilityDenied == nil {
		t.Fatal("windowctl.ErrAccessibilityDenied is nil; expected a typed sentinel re-exported from core")
	}
	if !errors.Is(ErrAccessibilityDenied, core.ErrAccessibilityDenied) {
		t.Fatalf("windowctl.ErrAccessibilityDenied must alias core.ErrAccessibilityDenied; got %v vs %v",
			ErrAccessibilityDenied, core.ErrAccessibilityDenied)
	}
}

// TestErrAccessibilityDeniedMessageGuidesUser pins the user-facing
// message so it stays actionable per requirements.md §11. The smoke
// script (scripts/smoke-darwin.sh) also greps for "Accessibility
// permission denied" in stderr, so this test is the unit-level twin of
// that integration assertion.
func TestErrAccessibilityDeniedMessageGuidesUser(t *testing.T) {
	msg := ErrAccessibilityDenied.Error()
	for _, want := range []string{
		"Accessibility permission denied",
		"System Settings",
		"Privacy & Security",
		"Accessibility",
		// Per spec slice 11: the error message MUST point users at the
		// new opt-in subcommand so the remediation is one copy-paste
		// away. The smoke script (scripts/smoke-darwin.sh) still
		// greps for "Accessibility permission denied" — pinned above.
		"windowctl permissions",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("ErrAccessibilityDenied message missing %q\nfull message: %q", want, msg)
		}
	}
}

// TestMoveSurfacesAccessibilityDeniedFromAdapter ensures the public
// Move path lets the typed sentinel bubble up unchanged when the
// adapter denies the operation. The CGO/AX surface itself is not
// unit-testable on macOS without a granted runtime, so we validate the
// pure-Go plumbing here and the smoke script validates the real path.
func TestMoveSurfacesAccessibilityDeniedFromAdapter(t *testing.T) {
	a := &mockAdapter{windows: sampleWindows(), moveErr: ErrAccessibilityDenied}
	err := moveWith(a, Match{Title: "chrome"}, Target{Bounds: Rect{X: 10, Y: 20, W: 100, H: 200}})
	if !errors.Is(err, ErrAccessibilityDenied) {
		t.Fatalf("expected ErrAccessibilityDenied to propagate through moveWith, got %v", err)
	}
}

// TestFocusSurfacesAccessibilityDeniedFromAdapter — same contract as
// the Move variant, but for the focus path.
func TestFocusSurfacesAccessibilityDeniedFromAdapter(t *testing.T) {
	a := &mockAdapter{windows: sampleWindows(), focusErr: ErrAccessibilityDenied}
	err := focusWith(a, Match{App: "Code"})
	if !errors.Is(err, ErrAccessibilityDenied) {
		t.Fatalf("expected ErrAccessibilityDenied to propagate through focusWith, got %v", err)
	}
}

// TestRequestAccessibilityDelegatesToAdapter pins the contract that the
// public RequestAccessibility() entry point is a thin pass-through to
// the platform adapter. Per spec slice 11 this is how the
// `windowctl permissions` subcommand reaches the macOS AX prompt while
// staying a no-op on linux/windows.
func TestRequestAccessibilityDelegatesToAdapter(t *testing.T) {
	a := &mockAdapter{}
	if err := requestAccessibilityWith(a); err != nil {
		t.Fatalf("expected nil error from delegate, got %v", err)
	}
	if a.requestAXCalled != 1 {
		t.Fatalf("expected adapter.RequestAccessibility to be called exactly once, got %d", a.requestAXCalled)
	}
}

// TestRequestAccessibilitySurfacesDeniedSentinel confirms the AX-denied
// path bubbles up unchanged so the CLI can produce the right exit code
// and message. Mirrors the Move/Focus sentinel-propagation tests.
func TestRequestAccessibilitySurfacesDeniedSentinel(t *testing.T) {
	a := &mockAdapter{requestAXErr: ErrAccessibilityDenied}
	err := requestAccessibilityWith(a)
	if !errors.Is(err, ErrAccessibilityDenied) {
		t.Fatalf("expected ErrAccessibilityDenied to propagate, got %v", err)
	}
}

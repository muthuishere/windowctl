package windowctl

import (
	"errors"
	"testing"
)

type mockAdapter struct {
	windows  []Window
	monitors []Monitor
	moved    map[string]Rect
	focused  string
	listErr  error
	moveErr  error
	focusErr error
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

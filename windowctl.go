// Package windowctl is the public Go API for the windowctl cross-platform
// window manager. It re-exports the OS-agnostic types from internal/core and
// delegates all operations to the platform adapter selected at build time.
package windowctl

import (
	"strings"

	"github.com/muthuishere/windowctl/internal/core"
)

type (
	Window  = core.Window
	Monitor = core.Monitor
	Rect    = core.Rect
	Filter  = core.Filter
	Match   = core.Match
	Target  = core.Target
	Adapter = core.Adapter
)

var (
	ErrNotImplemented = core.ErrNotImplemented
	ErrNoMatch        = core.ErrNoMatch
)

var defaultAdapter Adapter = newPlatformAdapter()

func ListWindows(filter Filter) ([]Window, error) {
	return listWindowsWith(defaultAdapter, filter)
}

func ListMonitors() ([]Monitor, error) {
	return defaultAdapter.ListMonitors()
}

func Move(match Match, target Target) error {
	return moveWith(defaultAdapter, match, target)
}

func Focus(match Match) error {
	return focusWith(defaultAdapter, match)
}

// MoveZone moves the window matched by `match` into the given zone on the
// resolved monitor. monitorID is optional: when nil, the monitor is
// auto-resolved as the one containing the majority of the window's area
// (FR-MOV-03). zoneStr accepts enum (e.g. "2B") or split (e.g. "3:1") form.
func MoveZone(match Match, monitorID *int, zoneStr string) error {
	return moveZoneWith(defaultAdapter, match, monitorID, zoneStr)
}

func moveZoneWith(a Adapter, match Match, monitorID *int, zoneStr string) error {
	zone, err := ParseZone(zoneStr)
	if err != nil {
		return err
	}
	w, err := matchOne(a, match)
	if err != nil {
		return err
	}
	monitors, err := a.ListMonitors()
	if err != nil {
		return err
	}
	var mon Monitor
	if monitorID != nil {
		mon, err = findMonitor(monitors, *monitorID)
		if err != nil {
			return err
		}
	} else {
		mon, err = resolveCurrentMonitor(w, monitors)
		if err != nil {
			return err
		}
	}
	return a.Move(w.ID, zone.Rect(mon))
}

func listWindowsWith(a Adapter, filter Filter) ([]Window, error) {
	ws, err := a.ListWindows()
	if err != nil {
		return nil, err
	}
	return applyFilter(ws, filter), nil
}

func moveWith(a Adapter, match Match, target Target) error {
	w, err := matchOne(a, match)
	if err != nil {
		return err
	}
	return a.Move(w.ID, target.Bounds)
}

func focusWith(a Adapter, match Match) error {
	w, err := matchOne(a, match)
	if err != nil {
		return err
	}
	return a.Focus(w.ID)
}

func matchOne(a Adapter, match Match) (Window, error) {
	ws, err := a.ListWindows()
	if err != nil {
		return Window{}, err
	}
	matches := applyFilter(ws, Filter{Title: match.Title, App: match.App})
	if len(matches) == 0 {
		return Window{}, ErrNoMatch
	}
	return matches[0], nil
}

func applyFilter(ws []Window, f Filter) []Window {
	if f.Title == "" && f.App == "" {
		return ws
	}
	out := make([]Window, 0, len(ws))
	for _, w := range ws {
		if f.Title != "" && !strings.Contains(strings.ToLower(w.Title), strings.ToLower(f.Title)) {
			continue
		}
		if f.App != "" && !strings.EqualFold(w.App, f.App) {
			continue
		}
		out = append(out, w)
	}
	return out
}

package windowctl

import "fmt"

// resolveCurrentMonitor returns the monitor containing the majority of the
// window's visible area, per FR-MOV-03. When no monitor overlaps the window
// at all (e.g. window is offscreen) the primary monitor is returned, falling
// back to the first listed monitor.
func resolveCurrentMonitor(w Window, monitors []Monitor) (Monitor, error) {
	if len(monitors) == 0 {
		return Monitor{}, fmt.Errorf("monitor: no monitors available")
	}
	best := -1
	var winner Monitor
	for _, m := range monitors {
		area := overlapArea(w.Bounds, m)
		if area > best {
			best = area
			winner = m
		}
	}
	if best > 0 {
		return winner, nil
	}
	for _, m := range monitors {
		if m.Primary {
			return m, nil
		}
	}
	return monitors[0], nil
}

// findMonitor returns the monitor with the given ID or an error if missing.
func findMonitor(ms []Monitor, id int) (Monitor, error) {
	for _, m := range ms {
		if m.ID == id {
			return m, nil
		}
	}
	return Monitor{}, fmt.Errorf("monitor: invalid monitor ID %d", id)
}

func overlapArea(r Rect, m Monitor) int {
	x1 := max(r.X, m.X)
	y1 := max(r.Y, m.Y)
	x2 := min(r.X+r.W, m.X+m.Width)
	y2 := min(r.Y+r.H, m.Y+m.Height)
	if x2 <= x1 || y2 <= y1 {
		return 0
	}
	return (x2 - x1) * (y2 - y1)
}

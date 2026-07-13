package windowctl

import "fmt"

// SetWindowState applies a window-state change to the matched window:
// minimize, maximize (fill its current monitor), toggle native
// fullscreen, or close. Minimize/fullscreen/close go through the
// adapter's AX path; maximize is composed here as a Move to the window's
// monitor bounds so the adapter's native-op surface stays minimal (and
// so "maximize" means the same fill-the-screen gesture on every OS).
func SetWindowState(match Match, op WindowOp) error {
	return setWindowStateWith(defaultAdapter, match, op)
}

func setWindowStateWith(a Adapter, match Match, op WindowOp) error {
	if op == WindowMaximize {
		w, err := matchOne(a, match)
		if err != nil {
			return err
		}
		monitors, err := listMonitorsWith(a)
		if err != nil {
			return err
		}
		mon, err := resolveCurrentMonitor(w, monitors)
		if err != nil {
			return err
		}
		return a.Move(w.ID, monitorRect(mon))
	}
	w, err := matchOne(a, match)
	if err != nil {
		return err
	}
	return a.WindowState(w.ID, op)
}

// ParseWindowOp maps a CLI verb to a WindowOp. Unknown verbs are a hard
// error so the CLI can reject typos rather than silently no-op.
func ParseWindowOp(verb string) (WindowOp, error) {
	switch verb {
	case "minimize":
		return WindowMinimize, nil
	case "maximize":
		return WindowMaximize, nil
	case "fullscreen":
		return WindowFullscreen, nil
	case "close":
		return WindowClose, nil
	default:
		return 0, fmt.Errorf("unknown window-state verb %q (want minimize|maximize|fullscreen|close)", verb)
	}
}

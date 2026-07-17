package windowctl

import "errors"

// ScrollOptions parameterizes Scroll. DX/DY are wheel amounts in LINE
// units (positive DY scrolls the content up / wheel-away, negative down;
// positive DX scrolls right). X/Y are an optional cursor target the
// wheel is delivered at — both nil means "scroll where the cursor
// already is". When set they follow the monitor-relative rule (relative
// when Monitor is non-nil, absolute otherwise), same as MouseClick.
type ScrollOptions struct {
	Monitor *int
	X, Y    *int
	DX, DY  int
}

// Scroll delivers wheel events. It moves the cursor to the target point
// first (apps route wheel events to the window under the cursor), so a
// scroll with an explicit point is self-contained.
func Scroll(opts ScrollOptions) error {
	return scrollWith(defaultAdapter, opts)
}

func scrollWith(a Adapter, opts ScrollOptions) error {
	if (opts.X == nil) != (opts.Y == nil) {
		return errors.New("scroll: x and y must be given together")
	}
	if opts.DX == 0 && opts.DY == 0 {
		return errors.New("scroll: dx and dy cannot both be zero")
	}
	var gx, gy int
	if opts.X == nil {
		var err error
		gx, gy, err = a.CursorPosition()
		if err != nil {
			return err
		}
	} else {
		var err error
		gx, gy, err = resolvePoint(a, opts.Monitor, *opts.X, *opts.Y)
		if err != nil {
			return err
		}
	}
	return a.Scroll(gx, gy, opts.DX, opts.DY)
}

// DragOptions parameterizes Drag: press Button at (FromX,FromY), move
// through interpolated points, release at (ToX,ToY). Both points follow
// the monitor-relative rule (relative when Monitor is non-nil). Drives
// drag-and-drop, sliders, and text selection.
type DragOptions struct {
	Monitor                *int
	FromX, FromY, ToX, ToY int
	Button                 MouseButton
}

// Drag performs a press-hold-move-release gesture.
func Drag(opts DragOptions) error {
	return dragWith(defaultAdapter, opts)
}

func dragWith(a Adapter, opts DragOptions) error {
	fx, fy, err := resolvePoint(a, opts.Monitor, opts.FromX, opts.FromY)
	if err != nil {
		return err
	}
	tx, ty, err := resolvePoint(a, opts.Monitor, opts.ToX, opts.ToY)
	if err != nil {
		return err
	}
	return a.Drag(fx, fy, tx, ty, opts.Button)
}

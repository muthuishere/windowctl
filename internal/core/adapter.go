package core

import "errors"

var ErrNotImplemented = errors.New("windowctl: operation not implemented on this platform")

var ErrNoMatch = errors.New("windowctl: no window matched the filter")

type Adapter interface {
	ListWindows() ([]Window, error)
	ListMonitors() ([]Monitor, error)
	Move(windowID string, bounds Rect) error
	Focus(windowID string) error
}

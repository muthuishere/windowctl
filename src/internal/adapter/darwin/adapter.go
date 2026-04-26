//go:build darwin

// Package darwin is the macOS platform adapter for windowctl.
//
// Os-spikes tracer-bullet status: this slice ships the per-OS package
// scaffolding and adapter wiring only — every method returns
// core.ErrNotImplemented. Wiring ListWindows/ListMonitors to CoreGraphics
// and Move/Focus to the Accessibility API (with CGO and the framework
// linker flags from §9.2) is a follow-up slice.
package darwin

import "github.com/muthuishere/windowctl/src/internal/core"

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ListWindows() ([]core.Window, error) {
	return nil, core.ErrNotImplemented
}

func (a *Adapter) ListMonitors() ([]core.Monitor, error) {
	return nil, core.ErrNotImplemented
}

func (a *Adapter) Move(id string, b core.Rect) error {
	return core.ErrNotImplemented
}

func (a *Adapter) Focus(id string) error {
	return core.ErrNotImplemented
}

// Package core holds the OS-agnostic types and adapter contract that all
// platform adapters and the public windowctl package depend on. It exists to
// break the import cycle that would otherwise form between the public package
// and the per-OS adapter implementations.
package core

type Window struct {
	ID         string
	Title      string
	App        string
	PID        int
	Executable string
	Monitor    int
	Bounds     Rect
}

type Monitor struct {
	ID      int
	X, Y    int
	Width   int
	Height  int
	Primary bool
	// Active is true when the OS cursor is currently over this
	// monitor. Independent of Focused — the user can mouse over one
	// display while typing into a window on another.
	Active bool
	// Focused is true when the frontmost (key) window's centroid
	// falls on this monitor. The "where am I working" signal as
	// opposed to Active's "where is my pointer".
	Focused bool
}

type Rect struct {
	X, Y int
	W, H int
}

type Filter struct {
	Title string
	App   string
}

type Match struct {
	Title string
	App   string
}

type Target struct {
	Monitor *int
	Bounds  Rect
}

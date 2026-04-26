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

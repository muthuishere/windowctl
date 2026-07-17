// Package core holds the OS-agnostic types and adapter contract that all
// platform adapters and the public windowctl package depend on. It exists to
// break the import cycle that would otherwise form between the public package
// and the per-OS adapter implementations.
package core

type Window struct {
	ID    string
	Title string
	App   string
	PID   int
	// Monitor is the 1-indexed ID of the monitor whose bounds contain
	// the window's centroid. 0 means "off-screen / on no monitor"
	// (the window's centroid does not lie within any reported display).
	// Stamped by the public listWindowsWith path, not by adapters.
	Monitor int
	// Focused is true when this window is the frontmost on the focused
	// monitor — i.e. the topmost (z-ordered) window whose centroid
	// lies on the monitor flagged Focused. Stamped by the public
	// listWindowsWith path.
	Focused bool
	Bounds  Rect
}

type Monitor struct {
	ID      int  `json:"ID"`
	X       int  `json:"X"`
	Y       int  `json:"Y"`
	Width   int  `json:"Width"`
	Height  int  `json:"Height"`
	Primary bool `json:"Primary"`
	// Active is true when the OS cursor is currently over this
	// monitor. Independent of Focused — the user can mouse over one
	// display while typing into a window on another.
	Active bool `json:"Active"`
	// Focused is true when the frontmost (key) window's centroid
	// falls on this monitor. The "where am I working" signal as
	// opposed to Active's "where is my pointer".
	Focused bool `json:"Focused"`
}

// Rect's JSON tags spell the dimensions out (Width/Height) so the wire
// format matches Monitor's. The Go field names stay terse (W/H) so
// existing callers using the public Rect struct don't break.
type Rect struct {
	X int `json:"X"`
	Y int `json:"Y"`
	W int `json:"Width"`
	H int `json:"Height"`
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

// TextMatch is one piece of on-screen text located by FindText (native
// OCR). Bounds and Click are absolute global points — the same space
// MouseClick consumes — so `find --text` output feeds `mouse click`
// with no transformation (the coordinate contract).
type TextMatch struct {
	Text       string `json:"text"`
	Confidence float64 `json:"confidence"`
	Bounds     Rect   `json:"bounds"`
	ClickX     int    `json:"click_x"`
	ClickY     int    `json:"click_y"`
}

// WindowOp names a window-state change applied by an adapter's
// WindowState. The public package parses the CLI verb into one of these.
type WindowOp int

const (
	WindowMinimize WindowOp = iota
	WindowMaximize
	WindowFullscreen
	WindowClose
)

//go:build windows

// visual_ext.go — stubs for the visual-loop extension verbs (OCR,
// clipboard, scroll/drag, window-state). darwin is the reference
// implementation (native Vision + NSPasteboard + AX); the Windows native
// path (SendInput wheel, OLE clipboard, Windows.Media.Ocr, ShowWindow)
// is a planned follow-up. Returning ErrNotImplemented keeps the CI
// matrix green — the interface is satisfied and the CLI surfaces an
// actionable "not implemented" error.
package windows

import "github.com/muthuishere/windowctl/internal/core"

func (a *Adapter) FindText(rect core.Rect) ([]core.TextMatch, error) {
	return nil, core.ErrNotImplemented
}

func (a *Adapter) Clipboard() (string, error) { return "", core.ErrNotImplemented }

func (a *Adapter) SetClipboard(text string) error { return core.ErrNotImplemented }

func (a *Adapter) Scroll(x, y, dx, dy int) error { return core.ErrNotImplemented }

func (a *Adapter) Drag(fromX, fromY, toX, toY int, button core.MouseButton) error {
	return core.ErrNotImplemented
}

func (a *Adapter) WindowState(id string, op core.WindowOp) error {
	return core.ErrNotImplemented
}

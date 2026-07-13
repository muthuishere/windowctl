// native_ext.go — Go bindings for the Objective-C helpers (Vision OCR,
// NSPasteboard). Kept in its own cgo translation unit so the Vision /
// AppKit framework links and ObjC headers stay off the pure-C files.
package darwin

/*
#cgo LDFLAGS: -framework Vision -framework AppKit -framework CoreGraphics -framework ApplicationServices
#include <stdlib.h>
#include "vision.h"
#include "pasteboard.h"
*/
import "C"

import (
	"unsafe"

	"github.com/muthuishere/windowctl/internal/core"
)

// ocrMax bounds how many recognized lines a single FindText capture
// returns — generous for any real screen, keeps the C buffer fixed.
const ocrMax = 512

// FindText captures the resolved global rect and runs native Vision OCR,
// returning one core.TextMatch per recognized line. Geometry is returned
// relative to the capture origin by the C helper; we add (rect.X,rect.Y)
// so bounds and click points are absolute global points — the same space
// MouseClick consumes. Substring filtering / confidence sorting is done
// by the public package (OS-agnostic).
func (a *Adapter) FindText(rect core.Rect) ([]core.TextMatch, error) {
	if err := inputAXGuard(); err != nil {
		// OCR needs Screen Recording, not AX — but capture-denied is
		// surfaced by the C call below; AX is not required for OCR, so
		// don't gate on it. (inputAXGuard intentionally NOT called.)
		_ = err
	}
	buf := make([]C.wctl_ocr_match, ocrMax)
	n := C.wctl_ocr_rect(
		C.double(rect.X), C.double(rect.Y),
		C.double(rect.W), C.double(rect.H),
		&buf[0], C.int(ocrMax))
	switch {
	case int(n) == -1:
		return nil, core.ErrScreenCaptureDenied
	case int(n) < 0:
		return nil, core.ErrNotImplemented
	}
	matches := make([]core.TextMatch, 0, int(n))
	for i := 0; i < int(n); i++ {
		m := buf[i]
		matches = append(matches, core.TextMatch{
			Text:       C.GoString(&m.text[0]),
			Confidence: float64(m.confidence),
			Bounds: core.Rect{
				X: rect.X + int(m.x), Y: rect.Y + int(m.y),
				W: int(m.w), H: int(m.h),
			},
			ClickX: rect.X + int(m.cx),
			ClickY: rect.Y + int(m.cy),
		})
	}
	return matches, nil
}

// Clipboard returns the system pasteboard's string contents.
func (a *Adapter) Clipboard() (string, error) {
	// Two-pass: size, then exact allocation if the first buffer truncated.
	const first = 4096
	buf := make([]C.char, first)
	n := int(C.wctl_clipboard_get(&buf[0], C.int(first)))
	if n < 0 {
		return "", nil // empty / non-string pasteboard → empty string
	}
	if n < first {
		return C.GoStringN(&buf[0], C.int(n)), nil
	}
	big := make([]C.char, n+1)
	n2 := int(C.wctl_clipboard_get(&big[0], C.int(n+1)))
	if n2 < 0 {
		return "", nil
	}
	if n2 > n {
		n2 = n
	}
	return C.GoStringN(&big[0], C.int(n2)), nil
}

// SetClipboard replaces the pasteboard contents.
func (a *Adapter) SetClipboard(text string) error {
	cs := C.CString(text)
	defer C.free(unsafe.Pointer(cs))
	if int(C.wctl_clipboard_set(cs)) != 0 {
		return core.ErrNotImplemented
	}
	return nil
}

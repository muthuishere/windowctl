//go:build darwin

// Package darwin is the macOS platform adapter for windowctl.
//
// ListWindows and ListMonitors are implemented against CoreGraphics; they
// require no special permissions beyond the Screen Recording prompt
// (window owner names / IDs / sizes always come back; titles are
// suppressed by the OS until the prompt is granted).
//
// Move and Focus stay stubbed because they need Accessibility (AX)
// permission, which is granted per-process and is not available on a
// fresh GitHub Actions runner. Wiring them to AXUIElementSetAttribute
// is the next slice of work for this OS.
package darwin

/*
#cgo LDFLAGS: -framework CoreGraphics -framework CoreFoundation -framework ApplicationServices

#include <CoreGraphics/CoreGraphics.h>
#include <CoreFoundation/CoreFoundation.h>
#include <stdint.h>
#include <stdlib.h>

// All option/enum bit-flag combinations live in C so the Go side never has
// to reason about CGWindowListOption typing or CFTypeRef pointer math.

static CFArrayRef wctl_list_windows(void) {
    return CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID
    );
}

// Returns 0/1 for "is this CFTypeRef non-null" so Go callers don't compare
// CFTypeRef-typed values directly.
static int wctl_cf_is_null(CFTypeRef r) {
    return r == NULL ? 1 : 0;
}

// Read a long-typed CFNumber out of a CFDictionary entry. Returns 0 if
// the entry is missing or not a CFNumber.
static long wctl_cf_long(CFDictionaryRef d, const char* key) {
    CFStringRef k = CFStringCreateWithCString(NULL, key, kCFStringEncodingUTF8);
    CFTypeRef v = CFDictionaryGetValue(d, k);
    CFRelease(k);
    if (v == NULL || CFGetTypeID(v) != CFNumberGetTypeID()) return 0;
    long out = 0;
    CFNumberGetValue((CFNumberRef)v, kCFNumberLongType, &out);
    return out;
}

// Read a CFString out of a CFDictionary entry as a freshly malloc'd UTF-8
// C string. Caller frees. Returns NULL when missing.
static char* wctl_cf_string_dup(CFDictionaryRef d, const char* key) {
    CFStringRef k = CFStringCreateWithCString(NULL, key, kCFStringEncodingUTF8);
    CFTypeRef v = CFDictionaryGetValue(d, k);
    CFRelease(k);
    if (v == NULL || CFGetTypeID(v) != CFStringGetTypeID()) return NULL;
    CFStringRef s = (CFStringRef)v;
    CFIndex len = CFStringGetLength(s);
    CFIndex max = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
    char* buf = (char*)malloc(max);
    if (!CFStringGetCString(s, buf, max, kCFStringEncodingUTF8)) {
        free(buf);
        return NULL;
    }
    return buf;
}

// Read CGRect from kCGWindowBounds (a CFDictionary describing X/Y/W/H).
static int wctl_cf_window_bounds(CFDictionaryRef d, double* x, double* y, double* w, double* h) {
    CFStringRef k = CFStringCreateWithCString(NULL, "kCGWindowBounds", kCFStringEncodingUTF8);
    CFTypeRef v = CFDictionaryGetValue(d, k);
    CFRelease(k);
    if (v == NULL || CFGetTypeID(v) != CFDictionaryGetTypeID()) return 0;
    CGRect r;
    if (!CGRectMakeWithDictionaryRepresentation((CFDictionaryRef)v, &r)) return 0;
    *x = r.origin.x;
    *y = r.origin.y;
    *w = r.size.width;
    *h = r.size.height;
    return 1;
}

// Returns count of active displays, or -1 on error. Pass NULL/0 to probe
// count, then call again with a pre-sized array to fill it.
static int wctl_list_displays(CGDirectDisplayID* out, int cap) {
    uint32_t count = 0;
    if (out == NULL) {
        if (CGGetActiveDisplayList(0, NULL, &count) != kCGErrorSuccess) return -1;
        return (int)count;
    }
    if (CGGetActiveDisplayList((uint32_t)cap, out, &count) != kCGErrorSuccess) return -1;
    return (int)count;
}

static CGDirectDisplayID wctl_main_display(void) {
    return CGMainDisplayID();
}

static void wctl_display_bounds(CGDirectDisplayID id, double* x, double* y, double* w, double* h) {
    CGRect r = CGDisplayBounds(id);
    *x = r.origin.x;
    *y = r.origin.y;
    *w = r.size.width;
    *h = r.size.height;
}
*/
import "C"

import (
	"fmt"
	"unsafe"

	"github.com/muthuishere/windowctl/internal/core"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ListWindows() ([]core.Window, error) {
	arr := C.wctl_list_windows()
	if C.wctl_cf_is_null(C.CFTypeRef(arr)) != 0 {
		return nil, fmt.Errorf("CGWindowListCopyWindowInfo returned null")
	}
	defer C.CFRelease(C.CFTypeRef(arr))

	n := int(C.CFArrayGetCount(arr))
	out := make([]core.Window, 0, n)
	for i := 0; i < n; i++ {
		raw := C.CFArrayGetValueAtIndex(arr, C.CFIndex(i))
		if raw == nil {
			continue
		}
		d := C.CFDictionaryRef(raw)

		w := core.Window{
			ID:    fmt.Sprintf("%d", int64(cfDictLong(d, "kCGWindowNumber"))),
			Title: cfDictString(d, "kCGWindowName"),
			App:   cfDictString(d, "kCGWindowOwnerName"),
			PID:   int(cfDictLong(d, "kCGWindowOwnerPID")),
		}

		var x, y, ww, hh C.double
		if C.wctl_cf_window_bounds(d, &x, &y, &ww, &hh) != 0 {
			w.Bounds = core.Rect{X: int(x), Y: int(y), W: int(ww), H: int(hh)}
		}
		out = append(out, w)
	}
	return out, nil
}

func (a *Adapter) ListMonitors() ([]core.Monitor, error) {
	count := int(C.wctl_list_displays(nil, 0))
	if count < 0 {
		return nil, fmt.Errorf("CGGetActiveDisplayList: count probe failed")
	}
	if count == 0 {
		return nil, nil
	}
	ids := make([]C.CGDirectDisplayID, count)
	got := int(C.wctl_list_displays(&ids[0], C.int(count)))
	if got < 0 {
		return nil, fmt.Errorf("CGGetActiveDisplayList: list failed")
	}
	main := C.wctl_main_display()
	out := make([]core.Monitor, 0, got)
	for i := 0; i < got; i++ {
		var x, y, w, h C.double
		C.wctl_display_bounds(ids[i], &x, &y, &w, &h)
		out = append(out, core.Monitor{
			ID:      i,
			X:       int(x),
			Y:       int(y),
			Width:   int(w),
			Height:  int(h),
			Primary: ids[i] == main,
		})
	}
	return out, nil
}

func (a *Adapter) Move(id string, b core.Rect) error {
	return core.ErrNotImplemented
}

func (a *Adapter) Focus(id string) error {
	return core.ErrNotImplemented
}

func cfDictLong(d C.CFDictionaryRef, key string) C.long {
	ckey := C.CString(key)
	defer C.free(unsafe.Pointer(ckey))
	return C.wctl_cf_long(d, ckey)
}

func cfDictString(d C.CFDictionaryRef, key string) string {
	ckey := C.CString(key)
	defer C.free(unsafe.Pointer(ckey))
	cstr := C.wctl_cf_string_dup(d, ckey)
	if cstr == nil {
		return ""
	}
	defer C.free(unsafe.Pointer(cstr))
	return C.GoString(cstr)
}

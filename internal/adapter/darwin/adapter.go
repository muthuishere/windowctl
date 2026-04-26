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
#include <string.h>

// Plain-old-data struct that Go can read directly. C owns the whole
// CoreFoundation memory dance; Go never touches a CFTypeRef.
typedef struct {
    long   id;
    long   pid;
    char*  title;       // malloc'd UTF-8; may be NULL.
    char*  app;         // malloc'd UTF-8; may be NULL.
    int    has_bounds;
    double x;
    double y;
    double w;
    double h;
} wctl_window_t;

typedef struct {
    int    id_index;     // assigned position in caller's list
    int    primary;      // 0/1
    double x;
    double y;
    double w;
    double h;
} wctl_monitor_t;

static long wctl_dict_long(CFDictionaryRef d, CFStringRef key) {
    CFTypeRef v = CFDictionaryGetValue(d, key);
    if (v == NULL || CFGetTypeID(v) != CFNumberGetTypeID()) return 0;
    long out = 0;
    CFNumberGetValue((CFNumberRef)v, kCFNumberLongType, &out);
    return out;
}

static char* wctl_dict_string(CFDictionaryRef d, CFStringRef key) {
    CFTypeRef v = CFDictionaryGetValue(d, key);
    if (v == NULL || CFGetTypeID(v) != CFStringGetTypeID()) return NULL;
    CFStringRef s = (CFStringRef)v;
    CFIndex len = CFStringGetLength(s);
    CFIndex max = CFStringGetMaximumSizeForEncoding(len, kCFStringEncodingUTF8) + 1;
    char* buf = (char*)malloc((size_t)max);
    if (!buf) return NULL;
    if (!CFStringGetCString(s, buf, max, kCFStringEncodingUTF8)) {
        free(buf);
        return NULL;
    }
    return buf;
}

static int wctl_dict_bounds(CFDictionaryRef d, CFStringRef key,
                            double* x, double* y, double* w, double* h) {
    CFTypeRef v = CFDictionaryGetValue(d, key);
    if (v == NULL || CFGetTypeID(v) != CFDictionaryGetTypeID()) return 0;
    CGRect r;
    if (!CGRectMakeWithDictionaryRepresentation((CFDictionaryRef)v, &r)) return 0;
    *x = r.origin.x;
    *y = r.origin.y;
    *w = r.size.width;
    *h = r.size.height;
    return 1;
}

// Collect all on-screen windows. Returns count (>= 0) on success or -1 on
// failure. On success *out is a malloc'd array of `count` entries; free
// with wctl_free_windows(out, count).
static int wctl_collect_windows(wctl_window_t** out) {
    *out = NULL;
    CFArrayRef arr = CGWindowListCopyWindowInfo(
        kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements,
        kCGNullWindowID
    );
    if (arr == NULL) return -1;

    CFIndex n = CFArrayGetCount(arr);
    wctl_window_t* ws = NULL;
    if (n > 0) {
        ws = (wctl_window_t*)calloc((size_t)n, sizeof(wctl_window_t));
        if (!ws) {
            CFRelease(arr);
            return -1;
        }
    }

    CFStringRef k_num    = CFSTR("kCGWindowNumber");
    CFStringRef k_pid    = CFSTR("kCGWindowOwnerPID");
    CFStringRef k_name   = CFSTR("kCGWindowName");
    CFStringRef k_owner  = CFSTR("kCGWindowOwnerName");
    CFStringRef k_bounds = CFSTR("kCGWindowBounds");

    for (CFIndex i = 0; i < n; i++) {
        const void* raw = CFArrayGetValueAtIndex(arr, i);
        if (raw == NULL) continue;
        CFDictionaryRef d = (CFDictionaryRef)raw;

        ws[i].id     = wctl_dict_long(d, k_num);
        ws[i].pid    = wctl_dict_long(d, k_pid);
        ws[i].title  = wctl_dict_string(d, k_name);
        ws[i].app    = wctl_dict_string(d, k_owner);
        ws[i].has_bounds = wctl_dict_bounds(
            d, k_bounds, &ws[i].x, &ws[i].y, &ws[i].w, &ws[i].h
        );
    }

    CFRelease(arr);
    *out = ws;
    return (int)n;
}

static void wctl_free_windows(wctl_window_t* ws, int count) {
    if (ws == NULL) return;
    for (int i = 0; i < count; i++) {
        if (ws[i].title) free(ws[i].title);
        if (ws[i].app)   free(ws[i].app);
    }
    free(ws);
}

// Collect all active monitors. Same memory ownership pattern as
// wctl_collect_windows but no nested allocations, so caller just calls
// free() on the returned pointer.
static int wctl_collect_monitors(wctl_monitor_t** out) {
    *out = NULL;
    uint32_t count = 0;
    if (CGGetActiveDisplayList(0, NULL, &count) != kCGErrorSuccess) return -1;
    if (count == 0) return 0;

    CGDirectDisplayID* ids = (CGDirectDisplayID*)calloc(count, sizeof(CGDirectDisplayID));
    if (!ids) return -1;
    if (CGGetActiveDisplayList(count, ids, &count) != kCGErrorSuccess) {
        free(ids);
        return -1;
    }

    wctl_monitor_t* ms = (wctl_monitor_t*)calloc(count, sizeof(wctl_monitor_t));
    if (!ms) { free(ids); return -1; }

    CGDirectDisplayID main = CGMainDisplayID();
    for (uint32_t i = 0; i < count; i++) {
        CGRect r = CGDisplayBounds(ids[i]);
        ms[i].id_index = (int)i;
        ms[i].primary = (ids[i] == main) ? 1 : 0;
        ms[i].x = r.origin.x;
        ms[i].y = r.origin.y;
        ms[i].w = r.size.width;
        ms[i].h = r.size.height;
    }
    free(ids);
    *out = ms;
    return (int)count;
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
	var raw *C.wctl_window_t
	n := int(C.wctl_collect_windows(&raw))
	if n < 0 {
		return nil, fmt.Errorf("CGWindowListCopyWindowInfo failed")
	}
	if n == 0 {
		return nil, nil
	}
	defer C.wctl_free_windows(raw, C.int(n))

	ws := unsafe.Slice(raw, n)
	out := make([]core.Window, 0, n)
	for i := 0; i < n; i++ {
		w := core.Window{
			ID:    fmt.Sprintf("%d", int64(ws[i].id)),
			PID:   int(ws[i].pid),
			Title: cStringOrEmpty(ws[i].title),
			App:   cStringOrEmpty(ws[i].app),
		}
		if ws[i].has_bounds != 0 {
			w.Bounds = core.Rect{
				X: int(ws[i].x),
				Y: int(ws[i].y),
				W: int(ws[i].w),
				H: int(ws[i].h),
			}
		}
		out = append(out, w)
	}
	return out, nil
}

func (a *Adapter) ListMonitors() ([]core.Monitor, error) {
	var raw *C.wctl_monitor_t
	n := int(C.wctl_collect_monitors(&raw))
	if n < 0 {
		return nil, fmt.Errorf("CGGetActiveDisplayList failed")
	}
	if n == 0 {
		return nil, nil
	}
	defer C.free(unsafe.Pointer(raw))

	ms := unsafe.Slice(raw, n)
	out := make([]core.Monitor, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, core.Monitor{
			ID:      int(ms[i].id_index),
			X:       int(ms[i].x),
			Y:       int(ms[i].y),
			Width:   int(ms[i].w),
			Height:  int(ms[i].h),
			Primary: ms[i].primary != 0,
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

func cStringOrEmpty(p *C.char) string {
	if p == nil {
		return ""
	}
	return C.GoString(p)
}

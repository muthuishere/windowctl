//go:build windows

// Package windows is the Windows platform adapter for windowctl.
//
// Implemented against user32.dll via golang.org/x/sys/windows. ListWindows
// enumerates top-level windows via EnumWindows; ListMonitors enumerates
// every attached display via EnumDisplayMonitors + GetMonitorInfoW.
package windows

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/muthuishere/windowctl/internal/core"
	"golang.org/x/sys/windows"
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextW      = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW   = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procMoveWindow          = user32.NewProc("MoveWindow")
	procSetForegroundWnd    = user32.NewProc("SetForegroundWindow")
	procGetWindowThreadPID  = user32.NewProc("GetWindowThreadProcessId")
	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	procGetCursorPos        = user32.NewProc("GetCursorPos")
	procGetForegroundWnd    = user32.NewProc("GetForegroundWindow")
)

type point struct{ X, Y int32 }

type rect struct{ Left, Top, Right, Bottom int32 }

// monitorInfo mirrors the Win32 MONITORINFO struct exactly:
//
//	typedef struct tagMONITORINFO {
//	    DWORD cbSize;     // must be set to sizeof(MONITORINFO) = 40
//	    RECT  rcMonitor;  // full display bounds (virtual-screen coords)
//	    RECT  rcWork;     // work-area (excludes taskbar etc.)
//	    DWORD dwFlags;    // bit 0 = MONITORINFOF_PRIMARY
//	} MONITORINFO;
//
// Total size: 4 + 16 + 16 + 4 = 40 bytes. We use plain MONITORINFO
// (not MONITORINFOEXW) because the spec only needs ID/X/Y/W/H/Primary —
// the device-name field would be dead weight.
type monitorInfo struct {
	CbSize    uint32
	RcMonitor rect
	RcWork    rect
	DwFlags   uint32
}

const monitorInfofPrimary uint32 = 1

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (a *Adapter) ListWindows() ([]core.Window, error) {
	var out []core.Window
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		visible, _, _ := procIsWindowVisible.Call(hwnd)
		if visible == 0 {
			return 1
		}
		titleLen, _, _ := procGetWindowTextLenW.Call(hwnd)
		if titleLen == 0 {
			return 1
		}
		buf := make([]uint16, titleLen+1)
		procGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), titleLen+1)
		title := windows.UTF16ToString(buf)

		var pid uint32
		procGetWindowThreadPID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))

		var r rect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r)))

		out = append(out, core.Window{
			ID:    fmt.Sprintf("%d", hwnd),
			Title: title,
			PID:   int(pid),
			Bounds: core.Rect{
				X: int(r.Left),
				Y: int(r.Top),
				W: int(r.Right - r.Left),
				H: int(r.Bottom - r.Top),
			},
		})
		return 1
	})
	ret, _, err := procEnumWindows.Call(cb, 0)
	if ret == 0 {
		return nil, fmt.Errorf("EnumWindows: %w", err)
	}
	return out, nil
}

// ListMonitors enumerates every attached display via Win32
// EnumDisplayMonitors, then resolves each HMONITOR's bounds and
// primary-flag with GetMonitorInfoW. IDs are sequential 0..N-1 in
// enumeration order — same shape the darwin adapter returns so the
// public package can treat all platforms uniformly.
func (a *Adapter) ListMonitors() ([]core.Monitor, error) {
	var out []core.Monitor
	var enumErr error
	cb := syscall.NewCallback(func(hmon, _ uintptr, _ *rect, _ uintptr) uintptr {
		mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
		ret, _, err := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&mi)))
		if ret == 0 {
			enumErr = fmt.Errorf("GetMonitorInfoW: %w", err)
			return 0 // stop enumeration on hard failure
		}
		out = append(out, core.Monitor{
			ID:      len(out),
			X:       int(mi.RcMonitor.Left),
			Y:       int(mi.RcMonitor.Top),
			Width:   int(mi.RcMonitor.Right - mi.RcMonitor.Left),
			Height:  int(mi.RcMonitor.Bottom - mi.RcMonitor.Top),
			Primary: mi.DwFlags&monitorInfofPrimary != 0,
		})
		return 1
	})
	ret, _, err := procEnumDisplayMonitors.Call(0, 0, cb, 0)
	if ret == 0 {
		if enumErr != nil {
			return nil, enumErr
		}
		return nil, fmt.Errorf("EnumDisplayMonitors: %w", err)
	}
	if enumErr != nil {
		return nil, enumErr
	}

	// Stamp Active (cursor) + Focused (foreground window centroid).
	// GetCursorPos / GetForegroundWindow / GetWindowRect are
	// stateless and require no special permissions on Windows, so
	// it's fine to do these on every list call.
	var p point
	if ret, _, _ := procGetCursorPos.Call(uintptr(unsafe.Pointer(&p))); ret != 0 {
		stampMonitorByPoint(out, int(p.X), int(p.Y), func(m *core.Monitor) { m.Active = true })
	}
	hwnd, _, _ := procGetForegroundWnd.Call()
	if hwnd != 0 {
		var r rect
		if ret, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ret != 0 {
			cx := int(r.Left) + (int(r.Right)-int(r.Left))/2
			cy := int(r.Top) + (int(r.Bottom)-int(r.Top))/2
			stampMonitorByPoint(out, cx, cy, func(m *core.Monitor) { m.Focused = true })
		}
	}
	return out, nil
}

// stampMonitorByPoint runs `set` against the first monitor whose
// bounds contain (x, y). Shared between cursor (Active) and
// foreground-centroid (Focused) probes.
func stampMonitorByPoint(ms []core.Monitor, x, y int, set func(*core.Monitor)) {
	for i := range ms {
		m := ms[i]
		if x >= m.X && x < m.X+m.Width && y >= m.Y && y < m.Y+m.Height {
			set(&ms[i])
			return
		}
	}
}

func (a *Adapter) Move(id string, b core.Rect) error {
	hwnd, err := parseHwnd(id)
	if err != nil {
		return err
	}
	ret, _, err := procMoveWindow.Call(hwnd, uintptr(b.X), uintptr(b.Y), uintptr(b.W), uintptr(b.H), 1)
	if ret == 0 {
		return fmt.Errorf("MoveWindow %s: %w", id, err)
	}
	return nil
}

func (a *Adapter) Focus(id string) error {
	hwnd, err := parseHwnd(id)
	if err != nil {
		return err
	}
	ret, _, err := procSetForegroundWnd.Call(hwnd)
	if ret == 0 {
		return fmt.Errorf("SetForegroundWindow %s: %w", id, err)
	}
	return nil
}

// RequestAccessibility is a no-op on Windows. The Win32 windowing APIs
// used by the adapter (SetForegroundWindow, MoveWindow) gate on session
// / privilege rather than a per-process trust grant comparable to
// macOS Accessibility. The CLI keeps the method on the interface so
// `windowctl permissions` exists on all three platforms; here it just
// returns nil and the CLI prints a "not required" message based on
// runtime.GOOS.
func (a *Adapter) RequestAccessibility() error { return nil }

// CheckAccessibility is the non-prompting sibling of RequestAccessibility.
// AX is a macOS-only concept; on Windows there is nothing to check, so
// we always report trusted=true.
func (a *Adapter) CheckAccessibility() bool { return true }

func parseHwnd(id string) (uintptr, error) {
	var n uint64
	if _, err := fmt.Sscanf(id, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid window id %q: %w", id, err)
	}
	return uintptr(n), nil
}

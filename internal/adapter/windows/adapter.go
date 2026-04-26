//go:build windows

// Package windows is the Windows platform adapter for windowctl.
//
// Implemented against user32.dll via golang.org/x/sys/windows. The os-spikes
// tracer bullet covers list/move/focus; richer monitor enumeration is a
// planned follow-up.
package windows

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/muthuishere/windowctl/internal/core"
	"golang.org/x/sys/windows"
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows        = user32.NewProc("EnumWindows")
	procGetWindowTextW     = user32.NewProc("GetWindowTextW")
	procGetWindowTextLenW  = user32.NewProc("GetWindowTextLengthW")
	procIsWindowVisible    = user32.NewProc("IsWindowVisible")
	procGetWindowRect      = user32.NewProc("GetWindowRect")
	procMoveWindow         = user32.NewProc("MoveWindow")
	procSetForegroundWnd   = user32.NewProc("SetForegroundWindow")
	procGetWindowThreadPID = user32.NewProc("GetWindowThreadProcessId")
	procGetSystemMetrics   = user32.NewProc("GetSystemMetrics")
)

type rect struct{ Left, Top, Right, Bottom int32 }

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

func (a *Adapter) ListMonitors() ([]core.Monitor, error) {
	const SM_CXSCREEN, SM_CYSCREEN = 0, 1
	w, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	h, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	return []core.Monitor{{
		ID: 0, X: 0, Y: 0, Width: int(w), Height: int(h), Primary: true,
	}}, nil
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

func parseHwnd(id string) (uintptr, error) {
	var n uint64
	if _, err := fmt.Sscanf(id, "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid window id %q: %w", id, err)
	}
	return uintptr(n), nil
}

package core

import "errors"

var ErrNotImplemented = errors.New("windowctl: operation not implemented on this platform")

var ErrNoMatch = errors.New("windowctl: no window matched the filter")

// ErrAccessibilityDenied is returned by the macOS adapter when a Move
// or Focus call is attempted while the running process is not trusted
// by the Accessibility (AX) subsystem. The message instructs the user
// how to grant access, per requirements.md §11. Other adapters may
// reuse this sentinel if a similar concept ever applies.
//
// Per docs/specs/11-permissions-subcommand.md the message also points
// users at the `windowctl permissions` opt-in subcommand so the
// remediation is one copy-paste away. The smoke harness still greps
// for the leading "Accessibility permission denied" substring.
var ErrAccessibilityDenied = errors.New("windowctl: Accessibility permission denied — run 'windowctl permissions' to grant, or grant manually in System Settings → Privacy & Security → Accessibility, then re-run")

type Adapter interface {
	ListWindows() ([]Window, error)
	ListMonitors() ([]Monitor, error)
	Move(windowID string, bounds Rect) error
	Focus(windowID string) error
	// RequestAccessibility asks the platform whether the current
	// process holds the privileges needed for Move / Focus. On macOS
	// it triggers the AX trust check WITH prompt enabled, surfacing
	// the system "wants to control your computer" dialog the first
	// time it is called from a given parent process. On Linux and
	// Windows it is a no-op that returns nil — kept on the interface
	// so the CLI surface is symmetric across platforms.
	//
	// Returns ErrAccessibilityDenied when the platform reports the
	// process is not trusted. Returns nil on success or on platforms
	// where no permission setup is required.
	RequestAccessibility() error
}

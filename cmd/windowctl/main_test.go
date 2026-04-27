package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	windowctl "github.com/muthuishere/windowctl"
)

func TestPrintWindowsTableHasSpecHeaderColumns(t *testing.T) {
	var buf bytes.Buffer
	printWindowsTable(&buf, []windowctl.Window{
		{ID: "1", Title: "Inbox", App: "Chrome", PID: 42, Monitor: 0, Bounds: windowctl.Rect{X: 0, Y: 0, W: 800, H: 600}},
	})
	out := buf.String()
	for _, col := range []string{"ID", "TITLE", "APP", "PID", "MONITOR", "BOUNDS"} {
		if !strings.Contains(out, col) {
			t.Errorf("expected column %q in table header, got:\n%s", col, out)
		}
	}
	if !strings.Contains(out, "800x600+0+0") {
		t.Errorf("expected bounds %q in row, got:\n%s", "800x600+0+0", out)
	}
}

func TestPrintWindowsJSONIsArrayOfObjects(t *testing.T) {
	var buf bytes.Buffer
	if err := printWindowsJSON(&buf, []windowctl.Window{
		{ID: "1", Title: "Inbox", App: "Chrome"},
	}); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not a JSON array: %v\n%s", err, buf.String())
	}
	if len(got) != 1 || got[0]["Title"] != "Inbox" {
		t.Fatalf("unexpected JSON shape: %s", buf.String())
	}
}

func TestPrintMonitorsTableHasSpecHeaderColumns(t *testing.T) {
	var buf bytes.Buffer
	printMonitorsTable(&buf, []windowctl.Monitor{
		{ID: 0, X: 0, Y: 0, Width: 1920, Height: 1080, Primary: true},
	})
	out := buf.String()
	for _, col := range []string{"ID", "X", "Y", "WIDTH", "HEIGHT", "PRIMARY"} {
		if !strings.Contains(out, col) {
			t.Errorf("expected column %q in table header, got:\n%s", col, out)
		}
	}
}

func TestPrintMonitorsJSONIsArrayOfObjects(t *testing.T) {
	var buf bytes.Buffer
	if err := printMonitorsJSON(&buf, []windowctl.Monitor{
		{ID: 0, Width: 1920, Height: 1080, Primary: true},
	}); err != nil {
		t.Fatal(err)
	}
	var got []map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not a JSON array: %v\n%s", err, buf.String())
	}
	if len(got) != 1 || got[0]["Primary"] != true {
		t.Fatalf("unexpected JSON shape: %s", buf.String())
	}
}

// TestUsageMentionsPermissionsSubcommand pins ac-1: the help / usage
// text MUST advertise the new permissions subcommand alongside
// windows / monitors / move / focus.
func TestUsageMentionsPermissionsSubcommand(t *testing.T) {
	var buf bytes.Buffer
	usage(&buf)
	if !strings.Contains(buf.String(), "windowctl permissions") {
		t.Fatalf("usage text missing 'windowctl permissions':\n%s", buf.String())
	}
}

// alwaysGranted / alwaysDenied are CheckAccessibility test stubs that
// also record whether they were invoked, so individual tests can pin
// "the --status path called check, not request".
type checkStub struct {
	result bool
	called int
}

func (s *checkStub) fn() bool {
	s.called++
	return s.result
}

type requestStub struct {
	err    error
	called int
}

func (s *requestStub) fn() error {
	s.called++
	return s.err
}

// TestRunPermissionsDarwinGranted exercises the macOS happy path: when
// the adapter's RequestAccessibility returns nil the command prints a
// "granted" line to stdout and exits 0. Per spec slice 11.
func TestRunPermissionsDarwinGranted(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := runPermissions(&stdout, &stderr, "darwin", false, func() error { return nil }, func() bool { return true })
	if rc != 0 {
		t.Fatalf("expected exit 0 on AX granted, got %d (stderr=%q)", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "granted") {
		t.Fatalf("expected stdout to mention 'granted', got %q", stdout.String())
	}
}

// TestRunPermissionsDarwinDenied exercises the macOS unhappy path: when
// the adapter returns ErrAccessibilityDenied the command MUST exit
// non-zero and the stderr message MUST mention System Settings so the
// user has the manual route as a fallback.
func TestRunPermissionsDarwinDenied(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := runPermissions(&stdout, &stderr, "darwin", false, func() error {
		return windowctl.ErrAccessibilityDenied
	}, func() bool { return false })
	if rc == 0 {
		t.Fatalf("expected non-zero exit on AX denied, got 0 (stdout=%q)", stdout.String())
	}
	if !strings.Contains(stderr.String(), "System Settings") {
		t.Fatalf("expected stderr to mention 'System Settings', got %q", stderr.String())
	}
}

// TestRunPermissionsLinuxIsNoOp pins ac-3: on linux the subcommand
// prints a "not required" line and exits 0. Same for windows below.
func TestRunPermissionsLinuxIsNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := runPermissions(&stdout, &stderr, "linux", false, func() error { return nil }, func() bool { return true })
	if rc != 0 {
		t.Fatalf("expected exit 0 on linux no-op, got %d", rc)
	}
	if !strings.Contains(stdout.String(), "not required") || !strings.Contains(stdout.String(), "linux") {
		t.Fatalf("expected stdout to mention 'not required on linux', got %q", stdout.String())
	}
}

func TestRunPermissionsWindowsIsNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	rc := runPermissions(&stdout, &stderr, "windows", false, func() error { return nil }, func() bool { return true })
	if rc != 0 {
		t.Fatalf("expected exit 0 on windows no-op, got %d", rc)
	}
	if !strings.Contains(stdout.String(), "not required") || !strings.Contains(stdout.String(), "windows") {
		t.Fatalf("expected stdout to mention 'not required on windows', got %q", stdout.String())
	}
}

// TestRunPermissionsUnexpectedError pins behaviour when the adapter
// returns a non-sentinel error (e.g. an internal CFDictionary failure
// surfaced as a generic error). The CLI MUST exit non-zero and surface
// the error rather than swallow it.
func TestRunPermissionsUnexpectedError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	boom := errors.New("ax check exploded")
	rc := runPermissions(&stdout, &stderr, "darwin", false, func() error { return boom }, func() bool { return false })
	if rc == 0 {
		t.Fatalf("expected non-zero exit on unexpected error, got 0")
	}
	if !strings.Contains(stderr.String(), "ax check exploded") {
		t.Fatalf("expected stderr to surface the underlying error, got %q", stderr.String())
	}
}

// TestRunPermissionsStatusDarwinGranted pins the read-only happy path:
// --status on darwin with AX trust granted prints "granted" and exits 0
// WITHOUT invoking RequestAccessibility (the prompting one).
func TestRunPermissionsStatusDarwinGranted(t *testing.T) {
	var stdout, stderr bytes.Buffer
	req := &requestStub{}
	chk := &checkStub{result: true}
	rc := runPermissions(&stdout, &stderr, "darwin", true, req.fn, chk.fn)
	if rc != 0 {
		t.Fatalf("expected exit 0 on AX granted, got %d (stderr=%q)", rc, stderr.String())
	}
	if !strings.Contains(stdout.String(), "granted") {
		t.Fatalf("expected stdout to mention 'granted', got %q", stdout.String())
	}
	if req.called != 0 {
		t.Fatalf("--status MUST NOT call RequestAccessibility (prompting); got %d calls", req.called)
	}
	if chk.called != 1 {
		t.Fatalf("--status MUST call CheckAccessibility exactly once; got %d", chk.called)
	}
}

// TestRunPermissionsStatusDarwinDenied pins the read-only unhappy path:
// --status on darwin with AX trust denied prints the actionable denial
// message to stderr (mentioning the manual route) and exits non-zero —
// again without invoking RequestAccessibility.
func TestRunPermissionsStatusDarwinDenied(t *testing.T) {
	var stdout, stderr bytes.Buffer
	req := &requestStub{}
	chk := &checkStub{result: false}
	rc := runPermissions(&stdout, &stderr, "darwin", true, req.fn, chk.fn)
	if rc == 0 {
		t.Fatalf("expected non-zero exit on AX denied, got 0 (stdout=%q)", stdout.String())
	}
	if !strings.Contains(stderr.String(), "denied") {
		t.Fatalf("expected stderr to mention 'denied', got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "System Settings") {
		t.Fatalf("expected stderr to mention 'System Settings', got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "windowctl permissions") {
		t.Fatalf("expected stderr to point at 'windowctl permissions' for remediation, got %q", stderr.String())
	}
	if req.called != 0 {
		t.Fatalf("--status MUST NOT call RequestAccessibility (prompting); got %d calls", req.called)
	}
	if chk.called != 1 {
		t.Fatalf("--status MUST call CheckAccessibility exactly once; got %d", chk.called)
	}
}

// TestRunPermissionsStatusLinuxIsNoOp pins ac-4: on linux --status
// prints a "not required" line, exits 0, and does NOT call either
// adapter method (the message is platform-derived, not adapter-derived).
func TestRunPermissionsStatusLinuxIsNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	req := &requestStub{}
	chk := &checkStub{result: true}
	rc := runPermissions(&stdout, &stderr, "linux", true, req.fn, chk.fn)
	if rc != 0 {
		t.Fatalf("expected exit 0 on linux --status, got %d", rc)
	}
	if !strings.Contains(stdout.String(), "not required") || !strings.Contains(stdout.String(), "linux") {
		t.Fatalf("expected stdout to mention 'not required on linux', got %q", stdout.String())
	}
	if req.called != 0 {
		t.Fatalf("--status on linux MUST NOT call RequestAccessibility; got %d calls", req.called)
	}
}

// TestRunPermissionsStatusWindowsIsNoOp — same as linux, for windows.
func TestRunPermissionsStatusWindowsIsNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	req := &requestStub{}
	chk := &checkStub{result: true}
	rc := runPermissions(&stdout, &stderr, "windows", true, req.fn, chk.fn)
	if rc != 0 {
		t.Fatalf("expected exit 0 on windows --status, got %d", rc)
	}
	if !strings.Contains(stdout.String(), "not required") || !strings.Contains(stdout.String(), "windows") {
		t.Fatalf("expected stdout to mention 'not required on windows', got %q", stdout.String())
	}
	if req.called != 0 {
		t.Fatalf("--status on windows MUST NOT call RequestAccessibility; got %d calls", req.called)
	}
}

// TestRunPermissionsNoStatusDarwinDoesNotCallCheck pins the symmetric
// guarantee: when --status is NOT set, the prompting RequestAccessibility
// path is taken and CheckAccessibility is not invoked.
func TestRunPermissionsNoStatusDarwinDoesNotCallCheck(t *testing.T) {
	var stdout, stderr bytes.Buffer
	req := &requestStub{}
	chk := &checkStub{result: true}
	rc := runPermissions(&stdout, &stderr, "darwin", false, req.fn, chk.fn)
	if rc != 0 {
		t.Fatalf("expected exit 0 on AX granted (no --status), got %d", rc)
	}
	if req.called != 1 {
		t.Fatalf("no-status path MUST call RequestAccessibility exactly once; got %d", req.called)
	}
	if chk.called != 0 {
		t.Fatalf("no-status path MUST NOT call CheckAccessibility; got %d", chk.called)
	}
}

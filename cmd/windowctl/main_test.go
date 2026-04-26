package main

import (
	"bytes"
	"encoding/json"
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

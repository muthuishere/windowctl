package windowctl

import (
	"strings"
	"testing"
)

func TestParseZoneAcceptsEnumCaseInsensitive(t *testing.T) {
	for _, in := range []string{"1A", "1a", "2D", "2d"} {
		z, err := ParseZone(in)
		if err != nil {
			t.Fatalf("ParseZone(%q): %v", in, err)
		}
		if z == nil {
			t.Fatalf("ParseZone(%q) returned nil", in)
		}
	}
}

func TestParseZoneRejectsInvalidEnum(t *testing.T) {
	for _, in := range []string{"3A", "1C", "foo", ""} {
		if _, err := ParseZone(in); err == nil {
			t.Fatalf("ParseZone(%q) expected error", in)
		}
	}
}

func TestParseZoneAcceptsValidSplits(t *testing.T) {
	for _, in := range []string{"2:1", "2:2", "3:1", "3:2", "3:3", "10:5"} {
		if _, err := ParseZone(in); err != nil {
			t.Fatalf("ParseZone(%q): %v", in, err)
		}
	}
}

func TestParseZoneRejectsInvalidSplits(t *testing.T) {
	for _, in := range []string{"0:1", "3:0", "3:4", "3:-1", "abc:1", "3:abc", ":", "3:"} {
		if _, err := ParseZone(in); err == nil {
			t.Fatalf("ParseZone(%q) expected error", in)
		}
	}
}

func mon(x, y, w, h int) Monitor { return Monitor{X: x, Y: y, Width: w, Height: h} }

func TestEnumZoneRectGivesExpectedHalvesAndQuadrants(t *testing.T) {
	m := mon(0, 0, 1920, 1080)

	cases := []struct {
		zone string
		want Rect
	}{
		{"1A", Rect{X: 0, Y: 0, W: 960, H: 1080}},
		{"1B", Rect{X: 960, Y: 0, W: 960, H: 1080}},
		{"2A", Rect{X: 0, Y: 0, W: 960, H: 540}},
		{"2B", Rect{X: 960, Y: 0, W: 960, H: 540}},
		{"2C", Rect{X: 0, Y: 540, W: 960, H: 540}},
		{"2D", Rect{X: 960, Y: 540, W: 960, H: 540}},
	}
	for _, c := range cases {
		z, _ := ParseZone(c.zone)
		got := z.Rect(m)
		if got != c.want {
			t.Errorf("zone %s: got %+v, want %+v", c.zone, got, c.want)
		}
	}
}

func TestEnumZoneRectRespectsMonitorOffset(t *testing.T) {
	m := mon(1920, 0, 1920, 1080)
	z, _ := ParseZone("2B")
	got := z.Rect(m)
	want := Rect{X: 1920 + 960, Y: 0, W: 960, H: 540}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestSplitZoneRectMatchesSpecFormula(t *testing.T) {
	m := mon(0, 0, 1500, 1000)

	cases := []struct {
		zone string
		want Rect
	}{
		{"3:1", Rect{X: 0, Y: 0, W: 500, H: 1000}},
		{"3:2", Rect{X: 500, Y: 0, W: 500, H: 1000}},
		{"3:3", Rect{X: 1000, Y: 0, W: 500, H: 1000}},
	}
	for _, c := range cases {
		z, _ := ParseZone(c.zone)
		got := z.Rect(m)
		if got != c.want {
			t.Errorf("zone %s: got %+v, want %+v", c.zone, got, c.want)
		}
	}
}

func TestSplitZoneRectRespectsMonitorOffset(t *testing.T) {
	m := mon(1920, 100, 1500, 1000)
	z, _ := ParseZone("3:2")
	got := z.Rect(m)
	want := Rect{X: 1920 + 500, Y: 100, W: 500, H: 1000}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestZoneStringRoundtrip(t *testing.T) {
	for _, in := range []string{"1A", "2D", "3:2", "10:5"} {
		z, err := ParseZone(in)
		if err != nil {
			t.Fatalf("ParseZone(%q): %v", in, err)
		}
		if !strings.EqualFold(z.String(), in) {
			t.Errorf("Zone.String() = %q, want %q", z.String(), in)
		}
	}
}

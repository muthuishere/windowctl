package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"text/tabwriter"

	windowctl "github.com/muthuishere/windowctl"
)

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}

	switch os.Args[1] {
	case "windows":
		windowsCmd(os.Args[2:])
	case "monitors":
		monitorsCmd(os.Args[2:])
	case "move":
		moveCmd(os.Args[2:])
	case "focus":
		focusCmd(os.Args[2:])
	case "-h", "--help", "help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w io.Writer) {
	fmt.Fprintln(w, `windowctl — cross-platform window management

Usage:
  windowctl windows list [--title <s>] [--app <s>] [--json]
  windowctl monitors list [--json]
  windowctl move (--title <s> | --app <s>) [--monitor <n>] (--zone <z> | --x <n> --y <n> --w <n> --h <n>)
  windowctl focus (--title <s> | --app <s>)`)
}

func windowsCmd(args []string) {
	if len(args) < 1 || args[0] != "list" {
		fmt.Fprintln(os.Stderr, "usage: windowctl windows list [--title <s>] [--app <s>] [--json]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("windows list", flag.ExitOnError)
	title := fs.String("title", "", "filter by window title (case-insensitive substring)")
	app := fs.String("app", "", "filter by application name (case-insensitive)")
	asJSON := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args[1:])

	ws, err := windowctl.ListWindows(windowctl.Filter{Title: *title, App: *app})
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
	if *asJSON {
		_ = printWindowsJSON(os.Stdout, ws)
		return
	}
	printWindowsTable(os.Stdout, ws)
}

func monitorsCmd(args []string) {
	if len(args) < 1 || args[0] != "list" {
		fmt.Fprintln(os.Stderr, "usage: windowctl monitors list [--json]")
		os.Exit(2)
	}
	fs := flag.NewFlagSet("monitors list", flag.ExitOnError)
	asJSON := fs.Bool("json", false, "emit JSON instead of a table")
	_ = fs.Parse(args[1:])

	ms, err := windowctl.ListMonitors()
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
	if *asJSON {
		_ = printMonitorsJSON(os.Stdout, ms)
		return
	}
	printMonitorsTable(os.Stdout, ms)
}

func printWindowsTable(out io.Writer, ws []windowctl.Window) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tTITLE\tAPP\tPID\tMONITOR\tBOUNDS")
	for _, w := range ws {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%d\t%dx%d+%d+%d\n",
			w.ID, w.Title, w.App, w.PID, w.Monitor, w.Bounds.W, w.Bounds.H, w.Bounds.X, w.Bounds.Y)
	}
	_ = tw.Flush()
}

func printWindowsJSON(out io.Writer, ws []windowctl.Window) error {
	return json.NewEncoder(out).Encode(ws)
}

func printMonitorsTable(out io.Writer, ms []windowctl.Monitor) {
	tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ID\tX\tY\tWIDTH\tHEIGHT\tPRIMARY")
	for _, m := range ms {
		fmt.Fprintf(tw, "%d\t%d\t%d\t%d\t%d\t%t\n", m.ID, m.X, m.Y, m.Width, m.Height, m.Primary)
	}
	_ = tw.Flush()
}

func printMonitorsJSON(out io.Writer, ms []windowctl.Monitor) error {
	return json.NewEncoder(out).Encode(ms)
}

func moveCmd(args []string) {
	fs := flag.NewFlagSet("move", flag.ExitOnError)
	title := fs.String("title", "", "match by window title (case-insensitive substring)")
	app := fs.String("app", "", "match by application name (case-insensitive)")
	monitor := fs.Int("monitor", -1, "target monitor ID (omit to auto-resolve)")
	zone := fs.String("zone", "", "target zone (1A,1B,2A..2D or N:M)")
	x := fs.Int("x", 0, "target x (with --monitor: relative; otherwise: absolute)")
	y := fs.Int("y", 0, "target y (with --monitor: relative; otherwise: absolute)")
	w := fs.Int("w", 0, "target width")
	h := fs.Int("h", 0, "target height")
	_ = fs.Parse(args)

	if *title == "" && *app == "" {
		fmt.Fprintln(os.Stderr, "windowctl move: --title or --app is required")
		os.Exit(2)
	}

	coordSet := false
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "x", "y", "w", "h":
			coordSet = true
		}
	})

	if *zone != "" && coordSet {
		fmt.Fprintln(os.Stderr, "windowctl move: --zone and --x/--y/--w/--h are mutually exclusive")
		os.Exit(2)
	}
	if *zone == "" && !coordSet {
		fmt.Fprintln(os.Stderr, "windowctl move: either --zone or --x/--y/--w/--h is required")
		os.Exit(2)
	}

	var monitorID *int
	if *monitor >= 0 {
		monitorID = monitor
	}

	match := windowctl.Match{Title: *title, App: *app}
	var err error
	if *zone != "" {
		err = windowctl.MoveZone(match, monitorID, *zone)
	} else {
		err = windowctl.MoveCoords(match, monitorID, windowctl.Rect{X: *x, Y: *y, W: *w, H: *h})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
}

func focusCmd(args []string) {
	fs := flag.NewFlagSet("focus", flag.ExitOnError)
	title := fs.String("title", "", "match by window title (case-insensitive substring)")
	app := fs.String("app", "", "match by application name (case-insensitive)")
	_ = fs.Parse(args)

	if *title == "" && *app == "" {
		fmt.Fprintln(os.Stderr, "windowctl focus: --title or --app is required")
		os.Exit(2)
	}

	if err := windowctl.Focus(windowctl.Match{Title: *title, App: *app}); err != nil {
		fmt.Fprintln(os.Stderr, "windowctl:", err)
		os.Exit(1)
	}
}

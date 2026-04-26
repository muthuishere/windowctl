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
  windowctl monitors list [--json]`)
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

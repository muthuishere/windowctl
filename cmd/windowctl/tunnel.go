// tunnel.go — cloudflared plumbing for `windowctl remote --tunnel`.
//
// This lives in the CLI on purpose: the library exposes a LAN-only
// server and cannot open a public tunnel at all. Going public is a
// choice a human makes at a command line, so the process that spawns
// cloudflared is the one a human started.
package main

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

// trycloudflareRe matches the public URL cloudflared prints to stderr.
var trycloudflareRe = regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)

// scanForTunnelURL reads cloudflared's stderr line by line and sends
// the first trycloudflare URL it sees, then drains the rest so the
// pipe never blocks the child.
func scanForTunnelURL(r io.Reader, out chan<- string) {
	sc := bufio.NewScanner(r)
	sent := false
	for sc.Scan() {
		if !sent {
			if m := trycloudflareRe.FindString(sc.Text()); m != "" {
				out <- strings.TrimRight(m, "/")
				sent = true
			}
		}
	}
}

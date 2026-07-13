// remote.go — `windowctl remote`: share the screen AND hand over
// mouse/keyboard control through a temporary cloudflared tunnel.
//
// This is a thin transport layer over the public automation API
// (Screenshot / MouseClick / TypeText / PressKey): a localhost HTTP
// server renders a viewer page that polls PNG frames and forwards
// clicks/keys back, and we spawn `cloudflared tunnel --url` to expose
// it, printing the *.trycloudflare.com URL. It puts no new
// window-management logic in the CLI — the library still owns every
// primitive.
//
// Security model (deliberate, because this grants desktop control to
// whoever opens the URL): the server binds 127.0.0.1 only, mints a
// fresh random token per run, and rejects every request without it.
// The token rides in the URL we print, so the link IS the key — share
// it like a house key and stop the server (Ctrl-C) to revoke. cloudflared
// terminates TLS and proxies to localhost; the token gates access.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	windowctl "github.com/muthuishere/windowctl"
)

func remoteCmd(args []string) {
	fs := flag.NewFlagSet("remote", flag.ExitOnError)
	monitor := fs.Int("monitor", 0, "monitor to share by default (>=1; omit for the focused one)")
	port := fs.Int("port", 0, "localhost port to bind (0 = pick a free one)")
	tunnel := fs.Bool("tunnel", false, "also expose a public URL via cloudflared (default: local URL only)")
	fps := fs.Float64("fps", 2, "max frames per second the viewer polls (1-10)")
	_ = fs.Parse(args)

	if *fps < 1 {
		*fps = 1
	}
	if *fps > 10 {
		*fps = 10
	}
	monitorID, err := monitorIDFromFlag(fs, monitor)
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl remote:", err)
		os.Exit(2)
	}

	srv := &remoteServer{
		defaultMonitor: monitorID,
		token:          mustToken(),
		frameInterval:  time.Duration(float64(time.Second) / *fps),
	}
	if rc := srv.run(*port, *tunnel); rc != 0 {
		os.Exit(rc)
	}
}

type remoteServer struct {
	defaultMonitor *int
	token          string
	frameInterval  time.Duration

	// screenshotMu serializes captures — the darwin capture path and
	// the temp-file it writes are not safe to run concurrently across
	// many polling viewers.
	screenshotMu sync.Mutex
}

func mustToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failing is unrecoverable and must never degrade
		// to a predictable token on a control channel.
		fmt.Fprintln(os.Stderr, "windowctl remote: cannot generate session token:", err)
		os.Exit(1)
	}
	return hex.EncodeToString(b)
}

func (s *remoteServer) run(port int, withTunnel bool) int {
	ln, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl remote: cannot bind localhost:", err)
		return 1
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port
	localURL := fmt.Sprintf("http://127.0.0.1:%d/?t=%s", actualPort, s.token)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.auth(s.handleIndex))
	mux.HandleFunc("/frame", s.auth(s.handleFrame))
	mux.HandleFunc("/monitors", s.auth(s.handleMonitors))
	mux.HandleFunc("/input", s.auth(s.handleInput))
	httpSrv := &http.Server{Handler: mux}

	go func() {
		if err := httpSrv.Serve(ln); err != nil && err != http.ErrServerClosed {
			fmt.Fprintln(os.Stderr, "windowctl remote: server error:", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Println("windowctl remote — screen share + CONTROL is live on this machine.")
	fmt.Printf("  Open this link (it carries the access token):\n\n    %s\n\n", localURL)

	var tunnel *exec.Cmd
	if withTunnel {
		tunnel, err = s.startTunnel(ctx, actualPort)
		if err != nil {
			fmt.Fprintln(os.Stderr, "windowctl remote:", err)
			_ = httpSrv.Close()
			return 1
		}
	}

	fmt.Println("Control is LIVE for anyone with the link. Press Ctrl-C to revoke and stop.")
	<-ctx.Done()

	fmt.Println("\nwindowctl remote: shutting down…")
	shutCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
	if tunnel != nil && tunnel.Process != nil {
		_ = tunnel.Process.Signal(syscall.SIGTERM)
		_ = tunnel.Wait()
	}
	return 0
}

// startTunnel spawns cloudflared and waits for it to print the public
// URL (which it writes to stderr). The token is appended so the link
// we surface is directly openable.
func (s *remoteServer) startTunnel(ctx context.Context, port int) (*exec.Cmd, error) {
	if _, err := exec.LookPath("cloudflared"); err != nil {
		return nil, fmt.Errorf("cloudflared not found on PATH — install it (brew install cloudflared) or use --no-tunnel")
	}
	cmd := exec.CommandContext(ctx, "cloudflared", "tunnel", "--url", fmt.Sprintf("http://127.0.0.1:%d", port))
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting cloudflared: %w", err)
	}

	urlCh := make(chan string, 1)
	go scanForTunnelURL(stderr, urlCh)

	select {
	case publicURL := <-urlCh:
		fmt.Printf("  Public link (cloudflared — carries the same token):\n\n    %s/?t=%s\n\n", publicURL, s.token)
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("cloudflared did not report a public URL within 30s")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return cmd, nil
}

func (s *remoteServer) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("t")
		if got == "" {
			got = r.Header.Get("X-Windowctl-Token")
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.token)) != 1 {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

func (s *remoteServer) handleMonitors(w http.ResponseWriter, r *http.Request) {
	ms, err := windowctl.ListMonitors()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ms)
}

// handleFrame captures the requested monitor and streams the PNG. The
// captured rect's global origin rides in headers so the viewer can map
// an image-pixel click back to a global coordinate (screenshots are
// point-normalized: origin + pixel == global point).
func (s *remoteServer) handleFrame(w http.ResponseWriter, r *http.Request) {
	monitorID := s.defaultMonitor
	if q := r.URL.Query().Get("monitor"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n >= 1 {
			monitorID = &n
		}
	}

	tmp, err := os.CreateTemp("", "windowctl-remote-*.png")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	s.screenshotMu.Lock()
	rect, err := windowctl.Screenshot(windowctl.ScreenshotOptions{Monitor: monitorID, OutPath: tmpPath})
	s.screenshotMu.Unlock()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := os.ReadFile(tmpPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Origin-X", strconv.Itoa(rect.X))
	w.Header().Set("X-Origin-Y", strconv.Itoa(rect.Y))
	w.Header().Set("X-Width", strconv.Itoa(rect.W))
	w.Header().Set("X-Height", strconv.Itoa(rect.H))
	_, _ = w.Write(data)
}

// inputAction is the wire shape the viewer POSTs. X/Y are ALWAYS
// absolute global coordinates (the viewer already added the frame
// origin), so the server passes them through with no monitor context.
type inputAction struct {
	Type   string `json:"type"` // "click" | "move" | "key" | "text"
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Button string `json:"button"` // "left" | "right" | "middle"
	Double bool   `json:"double"`
	Combo  string `json:"combo"` // for type "key"
	Text   string `json:"text"`  // for type "text"
}

func (s *remoteServer) handleInput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	var a inputAction
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, "bad json: "+err.Error(), http.StatusBadRequest)
		return
	}
	var err error
	switch a.Type {
	case "move":
		err = windowctl.MouseMove(nil, a.X, a.Y)
	case "click":
		x, y := a.X, a.Y
		opts := windowctl.ClickOptions{X: &x, Y: &y, Double: a.Double}
		switch a.Button {
		case "right":
			opts.Button = windowctl.MouseRight
		case "middle":
			opts.Button = windowctl.MouseMiddle
		}
		err = windowctl.MouseClick(opts)
	case "key":
		err = windowctl.PressKey(a.Combo)
	case "text":
		err = windowctl.TypeText(a.Text)
	default:
		http.Error(w, "unknown action type "+a.Type, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *remoteServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	buf.WriteString(remoteViewerHTML(s.token, int(s.frameInterval/time.Millisecond)))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(buf.Bytes())
}

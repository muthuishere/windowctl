// remote.go — `windowctl remote`: share the screen AND hand over
// mouse/keyboard control through a temporary cloudflared tunnel.
//
// This is a thin transport layer over the public automation API
// (Screenshot / MouseClick / TypeText / PressKey): an HTTP server
// streams the screen as live MJPEG (multipart/x-mixed-replace, JPEG
// frames pushed continuously — far smoother than single-frame polling)
// and forwards clicks/keys/touches back. It puts no new
// window-management logic in the CLI — the library still owns every
// primitive.
//
// Security model (deliberate, because this grants desktop control to
// whoever opens the URL): the server mints a fresh random token per run
// and rejects every request without it. The token rides in the URL we
// print, so the link IS the key — share it like a house key and stop
// the server (Ctrl-C) to revoke. By default it binds all interfaces so
// LAN devices (a phone, another laptop) can reach it directly; the
// token, not the bind address, is what gates access. `--tunnel` also
// spawns cloudflared for a public *.trycloudflare.com URL.
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
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
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
	fps := fs.Float64("fps", 10, "target frames per second for the live stream (1-30)")
	_ = fs.Parse(args)

	if *fps < 1 {
		*fps = 1
	}
	if *fps > 30 {
		*fps = 30
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
	// Bind all interfaces so other devices on the LAN can reach the
	// viewer by default (phones, another laptop) — the token gate is
	// what protects it, not the bind address. cloudflared (--tunnel)
	// stays an explicit opt-in for going beyond the LAN.
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		fmt.Fprintln(os.Stderr, "windowctl remote: cannot bind port:", err)
		return 1
	}
	actualPort := ln.Addr().(*net.TCPAddr).Port
	lanIP := lanIP()
	lanURL := fmt.Sprintf("http://%s:%d/?t=%s", lanIP, actualPort, s.token)
	loopbackURL := fmt.Sprintf("http://127.0.0.1:%d/?t=%s", actualPort, s.token)

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.auth(s.handleIndex))
	mux.HandleFunc("/stream", s.auth(s.handleStream))
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

	fmt.Println("windowctl remote — screen share + CONTROL is live.")
	fmt.Println("  Open from another device on your network (the link carries the access token):")
	fmt.Printf("\n    %s\n\n", lanURL)
	fmt.Printf("  On this machine: %s\n\n", loopbackURL)

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

// lanIP returns the machine's primary LAN IPv4 by opening a UDP socket
// toward a public address and reading the local address the OS picked
// (no packets are sent). Falls back to 127.0.0.1 when offline.
func lanIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
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

func (s *remoteServer) monitorFromQuery(r *http.Request) *int {
	monitorID := s.defaultMonitor
	if q := r.URL.Query().Get("monitor"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n >= 1 {
			monitorID = &n
		}
	}
	return monitorID
}

// captureJPEG captures the monitor to a JPEG (via the PNG the adapter
// writes, decoded and re-encoded). tmpPath is reused across a stream's
// frames so we don't churn temp files. quality is the JPEG quality
// (1-100). Returns the encoded bytes and the captured rect.
func (s *remoteServer) captureJPEG(monitorID *int, tmpPath string, quality int, buf *bytes.Buffer) (windowctl.Rect, error) {
	s.screenshotMu.Lock()
	rect, err := windowctl.Screenshot(windowctl.ScreenshotOptions{Monitor: monitorID, OutPath: tmpPath})
	s.screenshotMu.Unlock()
	if err != nil {
		return windowctl.Rect{}, err
	}
	raw, err := os.ReadFile(tmpPath)
	if err != nil {
		return windowctl.Rect{}, err
	}
	img, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		return windowctl.Rect{}, err
	}
	buf.Reset()
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return windowctl.Rect{}, err
	}
	return rect, nil
}

// handleStream is the live path: an MJPEG (multipart/x-mixed-replace)
// stream the browser renders natively in an <img>. This is what makes
// it feel live instead of the 2fps single-frame poll — frames are
// pushed continuously as JPEG (far smaller + faster than PNG) until the
// client disconnects. The viewer reads geometry from /monitors, so no
// per-frame headers are needed here.
func (s *remoteServer) handleStream(w http.ResponseWriter, r *http.Request) {
	monitorID := s.monitorFromQuery(r)
	quality := 60
	if q := r.URL.Query().Get("q"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n >= 10 && n <= 95 {
			quality = n
		}
	}
	interval := s.frameInterval
	if q := r.URL.Query().Get("fps"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n >= 1 && n <= 30 {
			interval = time.Second / time.Duration(n)
		}
	}

	tmp, err := os.CreateTemp("", "windowctl-stream-*.png")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+mw.Boundary())
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	var buf bytes.Buffer
	ctx := r.Context()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		rect, err := s.captureJPEG(monitorID, tmpPath, quality, &buf)
		if err != nil {
			return // client will retry the stream; a transient capture error ends this one
		}
		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Type", "image/jpeg")
		hdr.Set("Content-Length", strconv.Itoa(buf.Len()))
		hdr.Set("X-Origin-X", strconv.Itoa(rect.X))
		hdr.Set("X-Origin-Y", strconv.Itoa(rect.Y))
		hdr.Set("X-Width", strconv.Itoa(rect.W))
		hdr.Set("X-Height", strconv.Itoa(rect.H))
		part, err := mw.CreatePart(hdr)
		if err != nil {
			return
		}
		if _, err := part.Write(buf.Bytes()); err != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// handleFrame captures the requested monitor and streams the PNG. The
// captured rect's global origin rides in headers so the viewer can map
// an image-pixel click back to a global coordinate (screenshots are
// point-normalized: origin + pixel == global point). Kept as a
// single-frame fallback for clients that can't render MJPEG.
func (s *remoteServer) handleFrame(w http.ResponseWriter, r *http.Request) {
	monitorID := s.monitorFromQuery(r)

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

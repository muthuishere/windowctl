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
	key := fs.String("key", "", "shared access key gating every request (default: fresh random token per run). "+
		"Set a stable key when the viewer URL is stable — e.g. behind a named tunnel on your own domain — "+
		"so the link survives restarts. Falls back to $WINDOWCTL_REMOTE_KEY when the flag is empty.")
	_ = fs.Parse(args)

	// Resolve the access key: explicit --key wins, then the env var, then a
	// fresh random per-run token. A short key is a footgun on a control
	// channel that grants full desktop input, so require real entropy.
	accessKey := *key
	if accessKey == "" {
		accessKey = os.Getenv("WINDOWCTL_REMOTE_KEY")
	}
	if accessKey != "" && len(accessKey) < 16 {
		fmt.Fprintln(os.Stderr, "windowctl remote: --key must be at least 16 characters (it is the only thing gating desktop control)")
		os.Exit(2)
	}
	if accessKey == "" {
		accessKey = mustToken()
	}

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
		token:          accessKey,
		frameInterval:  time.Duration(float64(time.Second) / *fps),
		quality:        60,
	}
	if rc := srv.run(*port, *tunnel); rc != 0 {
		os.Exit(rc)
	}
}

type remoteServer struct {
	defaultMonitor *int
	token          string
	frameInterval  time.Duration
	quality        int

	// screenshotMu serializes captures — the darwin capture path and
	// its temp file are not concurrency-safe. With the shared feeds
	// below only one capture runs per monitor; the lock still guards
	// concurrent feeds for different monitors and the /frame fallback.
	screenshotMu sync.Mutex

	// feeds holds one shared capture loop per monitor, keyed by the
	// monitor query value ("" = default/focused). Every viewer of a
	// given monitor subscribes to the SAME feed rather than starting
	// its own capture — so N tabs / devices / reconnects of one monitor
	// never spin up N contending capture loops (which would thrash the
	// screenshot lock and cut everyone's framerate).
	feedsMu sync.Mutex
	feeds   map[string]*monitorFeed
}

// monitorFeed is a single shared capture loop for one monitor. Its
// latest JPEG frame is broadcast to all subscribers via cond. It is
// reference-counted: the loop starts on the first subscriber and stops
// when the last one leaves.
type monitorFeed struct {
	key       string
	monitorID *int
	srv       *remoteServer

	mu      sync.Mutex
	cond    *sync.Cond
	latest  []byte
	rect    windowctl.Rect
	seq     uint64 // bumped per new frame; 0 = none captured yet
	refs    int
	stopped bool
	err     error
}

// acquireFeed returns the shared feed for a monitor key, starting its
// capture loop on first use, and increments the refcount.
func (s *remoteServer) acquireFeed(key string, monitorID *int) *monitorFeed {
	s.feedsMu.Lock()
	defer s.feedsMu.Unlock()
	if s.feeds == nil {
		s.feeds = map[string]*monitorFeed{}
	}
	f, ok := s.feeds[key]
	if !ok {
		f = &monitorFeed{key: key, monitorID: monitorID, srv: s}
		f.cond = sync.NewCond(&f.mu)
		s.feeds[key] = f
		go f.captureLoop()
	}
	f.refs++
	return f
}

// releaseFeed drops a subscriber; when the last leaves it stops the
// capture loop (and wakes any straggler waiter so none block forever).
func (s *remoteServer) releaseFeed(f *monitorFeed) {
	s.feedsMu.Lock()
	last := false
	f.refs--
	if f.refs <= 0 {
		delete(s.feeds, f.key)
		last = true
	}
	s.feedsMu.Unlock()
	if last {
		f.mu.Lock()
		f.stopped = true
		f.cond.Broadcast()
		f.mu.Unlock()
	}
}

// captureLoop captures the monitor at the server frame interval and
// broadcasts each new frame. It exits once the feed is marked stopped.
func (f *monitorFeed) captureLoop() {
	tmp, err := os.CreateTemp("", "windowctl-feed-*.png")
	if err != nil {
		f.mu.Lock()
		f.err = err
		f.cond.Broadcast()
		f.mu.Unlock()
		return
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpPath)

	var buf bytes.Buffer
	ticker := time.NewTicker(f.srv.frameInterval)
	defer ticker.Stop()
	for {
		rect, cerr := f.srv.captureJPEG(f.monitorID, tmpPath, f.srv.quality, &buf)
		f.mu.Lock()
		if f.stopped {
			f.mu.Unlock()
			return
		}
		if cerr != nil {
			f.err = cerr
		} else {
			f.err = nil
			f.latest = append(f.latest[:0], buf.Bytes()...)
			f.rect = rect
			f.seq++
		}
		f.cond.Broadcast()
		f.mu.Unlock()
		<-ticker.C
	}
}

// waitNext blocks until a frame newer than lastSeq exists (or the feed
// stops / errors), returning a copy of the frame, its rect, and the new
// seq. A stopped feed returns http.ErrServerClosed.
func (f *monitorFeed) waitNext(lastSeq uint64) ([]byte, windowctl.Rect, uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for f.seq == lastSeq && f.err == nil && !f.stopped {
		f.cond.Wait()
	}
	if f.stopped {
		return nil, windowctl.Rect{}, lastSeq, http.ErrServerClosed
	}
	if f.err != nil && f.seq == lastSeq {
		return nil, windowctl.Rect{}, lastSeq, f.err
	}
	frame := make([]byte, len(f.latest))
	copy(frame, f.latest)
	return frame, f.rect, f.seq, nil
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
// stream the browser renders natively in an <img>. It subscribes to the
// SHARED per-monitor feed (acquireFeed) instead of capturing on its own,
// so any number of viewers of one monitor read a single capture loop —
// each frame captured once and fanned out. Geometry rides in per-part
// headers. The viewer also reads geometry from /monitors as a fallback.
func (s *remoteServer) handleStream(w http.ResponseWriter, r *http.Request) {
	monitorID := s.monitorFromQuery(r)
	key := r.URL.Query().Get("monitor") // "" == default/focused

	feed := s.acquireFeed(key, monitorID)
	defer s.releaseFeed(feed)

	mw := multipart.NewWriter(w)
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+mw.Boundary())
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "close")
	w.WriteHeader(http.StatusOK)

	flusher, _ := w.(http.Flusher)
	ctx := r.Context()
	var lastSeq uint64

	for {
		if ctx.Err() != nil {
			return
		}
		frame, rect, seq, err := feed.waitNext(lastSeq)
		if err != nil {
			return
		}
		lastSeq = seq
		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Type", "image/jpeg")
		hdr.Set("Content-Length", strconv.Itoa(len(frame)))
		hdr.Set("X-Origin-X", strconv.Itoa(rect.X))
		hdr.Set("X-Origin-Y", strconv.Itoa(rect.Y))
		hdr.Set("X-Width", strconv.Itoa(rect.W))
		hdr.Set("X-Height", strconv.Itoa(rect.H))
		part, err := mw.CreatePart(hdr)
		if err != nil {
			return
		}
		if _, err := part.Write(frame); err != nil {
			return
		}
		if flusher != nil {
			flusher.Flush()
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

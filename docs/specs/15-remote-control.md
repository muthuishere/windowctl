# Spec 15 — remote-control

- **Status:** Implemented
- **Source:** FR-RMT-01, §8.2, requirements §11
- **Summary:** `windowctl remote` serves a browser viewer that streams a monitor and forwards clicks/keys back, turning the automation primitives into a live remote-desktop session. Local URL by default; cloudflared public URL opt-in.

## Requirement: Local server + token gate

`windowctl remote` binds `127.0.0.1` on a free (or `--port`) port, mints a fresh random 128-bit token per run, and gates every endpoint (`/`, `/frame`, `/monitors`, `/input`) on it (query `?t=` or `X-Windowctl-Token` header, constant-time compared). The printed URL embeds the token, so **the link is the credential** — sharing it grants control, Ctrl-C revokes it (graceful `http.Server.Shutdown` + tunnel teardown).

#### Scenario: unauthenticated request

- **WHEN** any endpoint is hit without the token
- **THEN** it returns 403 and performs no capture or input.

## Requirement: Frame streaming (`/frame`)

`/frame?monitor=N` captures that monitor (default: the `--monitor` flag, else focused) via the public `Screenshot` and returns the PNG with the captured rect's global origin + size in `X-Origin-X/Y` and `X-Width/Height` headers. Captures are serialized behind a mutex (the darwin capture path + its temp file are not concurrency-safe across many polling viewers). The viewer polls at `--fps` (1–10, default 2).

## Requirement: Input forwarding (`/input`)

`POST /input` takes `{type: "click"|"move"|"key"|"text", ...}`. Coordinates are **already-absolute global points** — the browser maps an image-pixel click to a screen point client-side using the frame origin + the point-normalization guarantee (`screen = origin + naturalPixel`), so the server passes X/Y straight to `MouseMove`/`MouseClick` with no monitor context. `key` → `PressKey`, `text` → `TypeText`.

**Focus model:** the remote path intentionally uses the *unguarded* `TypeText`/`PressKey` (not `TypeInto`) — a live operator watching the streamed screen is the focus authority, exactly as with a physical keyboard. The FR-INP-02 focus guard is for headless scripting, not human teleoperation.

## Requirement: Optional public tunnel (`--tunnel`)

Without `--tunnel` the server is local-only (LAN reachable via the printed `127.0.0.1` URL, or the host's LAN IP). With `--tunnel`, `windowctl remote` spawns `cloudflared tunnel --url http://127.0.0.1:<port>`, scans its stderr for the `*.trycloudflare.com` URL (30s budget), and prints it with the token appended. cloudflared terminates TLS and proxies to localhost; the token still gates access. Missing `cloudflared` is a clear error pointing at `--no-tunnel`-equivalent local use.

## Implementation notes

- Lives in `cmd/windowctl/remote.go` + `remote_viewer.go` — a transport/runtime layer that composes the public API (`Screenshot`, `MouseClick`, `TypeText`, `PressKey`, `ListMonitors`), adding no window-management logic to the CLI. Consistent with the CLAUDE.md rule (business logic stays in the library; this is integration + an embedded single-page viewer).
- The viewer HTML is a template with placeholder substitution (not `fmt.Sprintf`) because the inline CSS/JS is full of literal `%`.
- macOS needs Accessibility (input) + Screen Recording (frames), same grants as the rest of automation.

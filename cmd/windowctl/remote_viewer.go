package main

import (
	"bufio"
	"io"
	"regexp"
	"strconv"
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

// remoteViewerHTML renders the single-page viewer. token is embedded so
// every /frame and /input request carries it; pollMS is the frame poll
// interval. The page maps a click on the displayed image back to the
// true screen point using the frame's natural pixel size and the
// origin headers (screenshots are point-normalized: 1 image pixel == 1
// screen point, so screenX = originX + naturalPixelX).
func remoteViewerHTML(token string, pollMS int) string {
	// Placeholder replacement (not Sprintf) because the embedded CSS/JS
	// is full of literal % signs. The token is server-minted hex, safe
	// to inline as a JS string literal.
	r := strings.NewReplacer(
		"__TOKEN__", token,
		"__POLL__", strconv.Itoa(pollMS),
	)
	return r.Replace(viewerTemplate)
}

const viewerTemplate = `<!doctype html>
<html>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>windowctl remote</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body { margin: 0; background: #0b0d10; color: #e6e6e6; font: 13px/1.4 system-ui, sans-serif; }
  header { display: flex; gap: 12px; align-items: center; padding: 8px 12px; background: #14171c; border-bottom: 1px solid #23272e; position: sticky; top: 0; z-index: 2; flex-wrap: wrap; }
  header b { color: #7cc4ff; }
  select, button { background: #1d2127; color: #e6e6e6; border: 1px solid #2c313a; border-radius: 6px; padding: 4px 8px; font: inherit; }
  button:hover { border-color: #3d84c6; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: #35c46a; display: inline-block; }
  .wrap { padding: 12px; display: flex; justify-content: center; }
  #screen { max-width: 100%; height: auto; border: 1px solid #23272e; border-radius: 6px; cursor: crosshair; box-shadow: 0 8px 40px rgba(0,0,0,.5); touch-action: none; -webkit-user-select: none; user-select: none; }
  #hint { color: #8a929e; }
  #status { color: #8a929e; margin-left: auto; }
  kbd { background: #23272e; border-radius: 4px; padding: 1px 5px; border: 1px solid #333; }
</style>
</head>
<body>
<header>
  <span class="dot"></span><b>windowctl remote</b>
  <label>Monitor <select id="mon"></select></label>
  <label><input type="checkbox" id="ctl" checked> allow control</label>
  <button id="kbd" type="button" title="show keyboard (touch devices)">⌨ keyboard</button>
  <span id="hint">tap / click = left · long-press or shift = right · double-tap = double-click</span>
  <span id="status">connecting…</span>
  <input id="kin" autocapitalize="off" autocomplete="off" autocorrect="off" spellcheck="false"
         style="position:fixed;opacity:0;pointer-events:none;bottom:0;left:0;width:1px;height:1px">
</header>
<div class="wrap"><img id="screen" alt="remote screen"></div>

<script>
const TOKEN = "__TOKEN__";
const POLL = __POLL__;
const img = document.getElementById('screen');
const monSel = document.getElementById('mon');
const ctl = document.getElementById('ctl');
const status = document.getElementById('status');
let originX = 0, originY = 0, natW = 0, natH = 0;
let monitor = '';
let mons = [];

// Geometry for click mapping comes from /monitors (not per-frame
// headers, which an MJPEG <img> stream can't expose to JS). Because
// captures are point-normalized, a monitor's Width/Height IS the frame's
// natural pixel size and its X/Y is the frame origin.
function applyGeometry() {
  const m = mons.find(x => String(x.ID) === String(monitor)) || mons[0];
  if (m) { originX = m.X; originY = m.Y; natW = m.Width; natH = m.Height; }
}

async function loadMonitors() {
  try {
    const r = await fetch('/monitors?t=' + TOKEN);
    mons = await r.json() || [];
    monSel.innerHTML = '';
    mons.forEach(m => {
      const o = document.createElement('option');
      o.value = m.ID;
      o.textContent = 'Monitor ' + m.ID + ' (' + m.Width + '×' + m.Height + ')' + (m.Focused ? ' • focused' : '');
      if (m.Focused) o.selected = true;
      monSel.appendChild(o);
    });
    monitor = monSel.value || '';
    applyGeometry();
  } catch (e) { /* single-monitor / race — stream still works */ }
}
monSel.addEventListener('change', () => { monitor = monSel.value; applyGeometry(); startStream(); });

// Live MJPEG stream: the browser renders multipart/x-mixed-replace
// natively in the <img>, so frames arrive pushed and continuous.
function startStream() {
  const u = '/stream?t=' + TOKEN + (monitor ? '&monitor=' + monitor : '') + '&_=' + Date.now();
  img.src = u;
  status.textContent = 'live • ' + natW + '×' + natH;
}
img.addEventListener('error', () => { status.textContent = 'reconnecting…'; setTimeout(startStream, 1000); });
img.addEventListener('load', () => { if (natW) status.textContent = 'live • ' + natW + '×' + natH; });

// toScreen maps a client (clientX, clientY) point — from either a mouse
// event or a touch point — to the true screen coordinate, using the
// displayed image rect and the point-normalization guarantee.
function toScreen(clientX, clientY) {
  const rect = img.getBoundingClientRect();
  const sx = (clientX - rect.left) / rect.width;
  const sy = (clientY - rect.top) / rect.height;
  return { x: originX + Math.round(sx * natW), y: originY + Math.round(sy * natH) };
}

async function send(action) {
  if (!ctl.checked) return;
  try {
    await fetch('/input?t=' + TOKEN, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(action),
    });
  } catch (e) {}
  // No manual refresh — the MJPEG stream updates the image continuously.
}

// --- Mouse control ---
img.addEventListener('click', ev => {
  const p = toScreen(ev.clientX, ev.clientY);
  send({ type: 'click', x: p.x, y: p.y, button: ev.shiftKey ? 'right' : 'left' });
});
img.addEventListener('dblclick', ev => {
  const p = toScreen(ev.clientX, ev.clientY);
  send({ type: 'click', x: p.x, y: p.y, button: 'left', double: true });
});
img.addEventListener('contextmenu', ev => {
  ev.preventDefault();
  const p = toScreen(ev.clientX, ev.clientY);
  send({ type: 'click', x: p.x, y: p.y, button: 'right' });
});

// --- Touch control (phone / tablet) ---
// A tap = left click. A long-press (>500ms without moving) = right
// click. A quick second tap within 300ms of the last = double-click.
// touchstart is preventDefault'd so the browser doesn't also synth a
// 300ms-late mouse 'click' (which would double-fire every tap).
let touchTimer = null, touchStart = null, lastTapAt = 0, movedFar = false;
img.addEventListener('touchstart', ev => {
  if (ev.touches.length !== 1) return;
  ev.preventDefault();
  const t = ev.touches[0];
  touchStart = { x: t.clientX, y: t.clientY };
  movedFar = false;
  touchTimer = setTimeout(() => {
    touchTimer = null;
    const p = toScreen(touchStart.x, touchStart.y);
    send({ type: 'click', x: p.x, y: p.y, button: 'right' });  // long-press → right-click
    touchStart = null;
  }, 500);
}, { passive: false });
img.addEventListener('touchmove', ev => {
  if (!touchStart || ev.touches.length !== 1) return;
  const t = ev.touches[0];
  if (Math.abs(t.clientX - touchStart.x) > 12 || Math.abs(t.clientY - touchStart.y) > 12) {
    movedFar = true;
    if (touchTimer) { clearTimeout(touchTimer); touchTimer = null; }
  }
}, { passive: true });
img.addEventListener('touchend', ev => {
  if (touchTimer) { clearTimeout(touchTimer); touchTimer = null; }
  if (!touchStart || movedFar) { touchStart = null; return; }
  ev.preventDefault();
  const p = toScreen(touchStart.x, touchStart.y);
  const now = Date.now();
  const isDouble = (now - lastTapAt) < 300;
  lastTapAt = now;
  send({ type: 'click', x: p.x, y: p.y, button: 'left', double: isDouble });
  touchStart = null;
}, { passive: false });

// Named keys and chords go through /input type:"key"; plain printable
// characters go through type:"text" so unicode is preserved.
const NAMED = { 'Enter':'enter','Tab':'tab','Escape':'esc','Backspace':'backspace','Delete':'delete',
  'ArrowUp':'up','ArrowDown':'down','ArrowLeft':'left','ArrowRight':'right',
  'Home':'home','End':'end','PageUp':'pageup','PageDown':'pagedown',' ':'space' };
// --- Mobile soft keyboard ---
// Physical keyboards fire keydown on document (handled below). Touch
// devices have no physical keyboard, so the "⌨ keyboard" button focuses
// a hidden input to raise the OS soft keyboard; its keydown gives us
// Enter/Backspace/arrows, and its 'input' event gives us typed glyphs
// (which we forward as text and then clear so it never accumulates).
const kin = document.getElementById('kin');
document.getElementById('kbd').addEventListener('click', () => {
  kin.style.pointerEvents = 'auto';
  kin.focus();
});
kin.addEventListener('input', () => {
  const v = kin.value;
  if (v) { send({ type: 'text', text: v }); kin.value = ''; }
});
kin.addEventListener('keydown', ev => {
  if (NAMED[ev.key] && ev.key !== ' ') {   // space comes through 'input'
    ev.preventDefault();
    send({ type: 'key', combo: NAMED[ev.key] });
  }
});

document.addEventListener('keydown', ev => {
  if (!ctl.checked) return;
  if (ev.target === kin || ev.target.tagName === 'SELECT') return;
  if (ev.target.tagName === 'INPUT') return;
  const mods = [];
  if (ev.metaKey) mods.push('cmd');
  if (ev.ctrlKey) mods.push('ctrl');
  if (ev.altKey) mods.push('alt');
  if (ev.shiftKey) mods.push('shift');
  let key = null;
  if (NAMED[ev.key]) key = NAMED[ev.key];
  else if (ev.key.length === 1) key = ev.key.toLowerCase();
  if (!key) return;
  // A bare printable char with no non-shift modifier: type it literally
  // so we don't lose the exact glyph to keycode mapping.
  const onlyShift = mods.length === 0 || (mods.length === 1 && mods[0] === 'shift');
  if (ev.key.length === 1 && onlyShift && !NAMED[ev.key]) {
    ev.preventDefault();
    send({ type: 'text', text: ev.key });
    return;
  }
  ev.preventDefault();
  send({ type: 'key', combo: mods.concat([key]).join('+') });
});

loadMonitors().then(startStream);
</script>
</body>
</html>`

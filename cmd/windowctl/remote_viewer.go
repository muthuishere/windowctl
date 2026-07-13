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
  #screen { max-width: 100%; height: auto; border: 1px solid #23272e; border-radius: 6px; cursor: crosshair; box-shadow: 0 8px 40px rgba(0,0,0,.5); }
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
  <span id="hint">click = left · shift+click = right · type to send keys · <kbd>Esc</kbd> Enter Tab arrows mapped</span>
  <span id="status">connecting…</span>
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

async function loadMonitors() {
  try {
    const r = await fetch('/monitors?t=' + TOKEN);
    const ms = await r.json();
    monSel.innerHTML = '';
    (ms || []).forEach(m => {
      const o = document.createElement('option');
      o.value = m.ID;
      o.textContent = 'Monitor ' + m.ID + ' (' + m.Width + '×' + m.Height + ')' + (m.Focused ? ' • focused' : '');
      if (m.Focused) o.selected = true;
      monSel.appendChild(o);
    });
    monitor = monSel.value || '';
  } catch (e) { /* single-monitor / race — frame still works */ }
}
monSel.addEventListener('change', () => { monitor = monSel.value; });

async function pullFrame() {
  try {
    const u = '/frame?t=' + TOKEN + (monitor ? '&monitor=' + monitor : '') + '&_=' + Date.now();
    const r = await fetch(u);
    if (!r.ok) { status.textContent = 'frame error: ' + r.status; return; }
    originX = parseInt(r.headers.get('X-Origin-X') || '0', 10);
    originY = parseInt(r.headers.get('X-Origin-Y') || '0', 10);
    natW = parseInt(r.headers.get('X-Width') || '0', 10);
    natH = parseInt(r.headers.get('X-Height') || '0', 10);
    const blob = await r.blob();
    const url = URL.createObjectURL(blob);
    const old = img.src;
    img.src = url;
    if (old.startsWith('blob:')) URL.revokeObjectURL(old);
    status.textContent = 'live • ' + natW + '×' + natH;
  } catch (e) { status.textContent = 'disconnected'; }
}

function toScreen(ev) {
  const rect = img.getBoundingClientRect();
  const sx = (ev.clientX - rect.left) / rect.width;
  const sy = (ev.clientY - rect.top) / rect.height;
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
  pullFrame();
}

img.addEventListener('click', ev => {
  const p = toScreen(ev);
  send({ type: 'click', x: p.x, y: p.y, button: ev.shiftKey ? 'right' : 'left' });
});
img.addEventListener('dblclick', ev => {
  const p = toScreen(ev);
  send({ type: 'click', x: p.x, y: p.y, button: 'left', double: true });
});
img.addEventListener('contextmenu', ev => {
  ev.preventDefault();
  const p = toScreen(ev);
  send({ type: 'click', x: p.x, y: p.y, button: 'right' });
});

// Named keys and chords go through /input type:"key"; plain printable
// characters go through type:"text" so unicode is preserved.
const NAMED = { 'Enter':'enter','Tab':'tab','Escape':'esc','Backspace':'backspace','Delete':'delete',
  'ArrowUp':'up','ArrowDown':'down','ArrowLeft':'left','ArrowRight':'right',
  'Home':'home','End':'end','PageUp':'pageup','PageDown':'pagedown',' ':'space' };
document.addEventListener('keydown', ev => {
  if (!ctl.checked) return;
  if (ev.target.tagName === 'SELECT' || ev.target.tagName === 'INPUT') return;
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

loadMonitors().then(pullFrame);
setInterval(pullFrame, POLL);
</script>
</body>
</html>`

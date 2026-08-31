---
title: Keyboard & mouse
description: Inject real input events — text, chords, clicks, scroll and drag.
---

```sh
windowctl type "hello world"
windowctl key cmd+shift+t
windowctl click --x 400 --y 300 --button left --count 2
windowctl move-mouse --x 960 --y 540
windowctl scroll --dy -120
windowctl drag --x1 100 --y1 100 --x2 600 --y2 400
```

These are real OS-level events (CGEvent on macOS), delivered to whatever has
focus — not messages posted to a specific window. Combine with `focus` when you
need to be sure where they land:

```sh
windowctl focus --app "Notes" && windowctl type "meeting notes"
```

## The visual loop

Screenshot, decide, act — the loop an agent uses to drive a UI it has no API
for:

```sh
windowctl screenshot --out /tmp/s.png     # look
windowctl click --x 812 --y 344           # act
windowctl screenshot --out /tmp/s2.png    # confirm
```

## As a stream

For continuous control, `windowctl start input --port N --token T` accepts input
events over a WebSocket instead of one process per action — which is what you
want when the events are coming from a test harness at speed.

## Permission

Input injection needs macOS Accessibility, same as moving windows. Availability
is reported honestly: `windowctl devices` tells you whether the input surface is
usable *before* you script against it.

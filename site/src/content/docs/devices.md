---
title: The device layer
description: A headless robot body for auto-testing apps that use the mic, camera, screen and keyboard.
---

The same binary that moves windows can also **be** the devices an application
under test talks to: type into it, click it, watch its screen, capture what it
plays, and feed it a microphone.

It exists because auto-testing a voice or video product needs something that can
hold a conversation with it — not a file fixture.

## Two ways to drive it

```sh
# one-shot verbs, no server
windowctl type "hello"
windowctl screenshot --out shot.png

# a control plane, for streams
windowctl serve --port 9099 --token "$TOKEN"
windowctl start video-out --port 9100 --token "$TOKEN"
```

`serve` speaks JSON over WebSocket: `hello` (token), then `list`, `start`,
`stop`. Starting a stream binds a fresh loopback port on demand and mints a
one-time **ticket** that the media connection must present. `start` is the
simpler path — a fixed port, authenticated by the token itself.

## Security model

- Every listener binds `127.0.0.1`. Never `0.0.0.0`, on any path.
- No token, no start: the server refuses rather than opening an unauthenticated
  port.
- Control connections must present a valid token in their first message, or they
  are closed.

## Frames

Media travels as a fixed **16-byte binary header** plus a raw payload — magic,
stream kind, format tag, sequence and timestamp — sized so a frame can be parsed
without allocating and forwarded with a single write. Raw and JPEG, not a codec:
deterministic bytes matter more than bandwidth on loopback, where a 10 ms audio
frame round-trips in 0.047 ms.

## Honest status

macOS only, today. Input, screenshot, screen capture and audio-device control
are working; speaker loopback delivers frames but needs the system-audio
permission to be non-silent; the virtual microphone driver is built but needs a
notarized Developer ID build to load; the virtual camera is designed, not built.
Windows parity is pending — note that **window control** is fully supported on
Windows regardless.

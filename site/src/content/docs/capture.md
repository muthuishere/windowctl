---
title: Screen & audio
description: Screenshots, screen capture, audio devices, and the virtual audio driver.
---

## Screenshots

```sh
windowctl screenshot --out shot.png
windowctl screenshot --display 2 --out external.png
windowctl screenshot --window 24815 --out chrome.png
```

ScreenCaptureKit on macOS: a cold 1920×1080 one-shot measures **110 ms**, most
of which is session setup. For continuous capture use the stream, where the
per-frame cost is far lower.

## Screen capture as a stream

```sh
windowctl start video-out --port 9100 --token "$TOKEN"
```

A persistent capture session pushes BGRA frames with the 16-byte header. Needs
the Screen Recording permission.

## Audio devices

```sh
windowctl audio list
windowctl audio default-in "windowctl Audio"
windowctl audio default-out "MacBook Pro Speakers"
```

Setting a default is done through CoreAudio directly, and it is **reversible** —
the prior default is restored when the run stops. A test that leaves your mic
pointed somewhere strange is a bug, not a side effect.

## Speaker loopback

```sh
windowctl start audio-out --port 9100 --token "$TOKEN"
```

Captures what the machine is playing, using the macOS process-tap and aggregate
device API — **no third-party audio driver required**. Frames flow today;
non-silent audio needs the system-audio capture permission, without which macOS
returns silence rather than an error. (That trap is documented in the repo's
spike results, because "samples flowed" looked like success until the peak level
was measured at exactly zero.)

## Our own virtual audio device

`native/audio-plugin/` is a CoreAudio AudioServerPlugIn written for this project
— a loopback device named **windowctl Audio** that is simultaneously a virtual
speaker and a virtual microphone: whatever an app plays into its output stream
is read back, sample for sample, from its input stream.

- User-space (`coreaudiod` loads it), no kernel extension.
- 2ch Float32, 44.1 and 48 kHz.
- The IO callback does `memcpy` against a pre-allocated ring buffer — no locks,
  no allocation, no socket I/O on the audio thread.
- Written fresh following Apple's permissively-licensed sample. Not derived from
  any GPL driver.

```sh
windowctl install audio     # build, sign, install, restart coreaudiod
```

Loading it requires a notarized Developer ID build — `coreaudiod` rejects
anything less, which is the gate this piece is currently behind.

---
title: How it is built
description: Layers, platform backends, and the spec-before-code process.
---

## Layers

1. **The command** — flag parsing and output formatting. No OS calls, no logic.
2. **The portable package** — zone maths, monitor resolution by majority
   overlap, filter matching, the coordinate rules. Unit tested once, identical
   everywhere.
3. **A tiny per-OS adapter** — six primitives, selected at link time by build
   tag. Anything composable from those primitives is deliberately kept *out* of
   here.
4. **The device layer** — protocol, input, screen, audio, and the stream
   servers, each with a native shim and a clean not-supported path off macOS.

## Platform backends

| Platform | Backend                                            |
|----------|----------------------------------------------------|
| macOS    | CoreGraphics + Accessibility API (CGO, no AppleScript, no private SPI) |
| Windows  | Win32 via `user32.dll`                             |
| Linux    | `wmctrl` + `xrandr`; native X11 planned            |
| Wayland  | Best-effort                                        |

## Process

Every capability follows the same order: a **runnable spike** that proves the
mechanism on real hardware, an **ADR** recording the decision and the rejected
alternatives, an **OpenSpec** slice describing the behaviour, and only then the
implementation.

The spikes are in the repository and they include the negative results — the
audio loopback that delivered a quarter of a million samples with a peak level
of exactly zero is written up as a trap, not deleted. Decisions that were
deferred (a kernel-mode virtual microphone on Windows, for instance) say so and
say why.

## Testing

The portable layer is unit tested. The adapters are validated by real-window
smoke tests in CI — launching an actual application on macOS and Windows
runners, moving it, and asserting the bounds it landed at, with the permission
state auto-detected so a fresh runner asserts the denied path instead of
failing.

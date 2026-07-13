# coordinate-contract

## ADDED Requirements

### Requirement: One documented global point coordinate space

`windowctl` SHALL document a single authoritative coordinate contract that all
point/rect-consuming and -producing verbs obey, so an agent can read a coordinate from one
verb's output and feed it to another without transformation.

#### Scenario: 1 image pixel == 1 global point

- **WHEN** any capture (`screenshot`, and the capture inside `find`) is produced on a
  HiDPI/Retina display
- **THEN** the image is resampled so one image pixel equals one global coordinate point,
  and a pixel at image `(px,py)` of a capture with origin `(ox,oy)` maps to global
  `(ox+px, oy+py)`.

#### Scenario: Consistency across verbs

- **WHEN** `find --text` returns a `click` point and it is passed to `mouse click`
- **THEN** the click lands on the located element, with no per-display or per-DPI
  correction required by the caller.

#### Scenario: Monitor-relative rule

- **WHEN** `--monitor N` is supplied to a point-consuming verb (mouse/scroll/drag/coords)
- **THEN** x/y are interpreted relative to monitor N's origin; absolute global otherwise.

#### Scenario: Negative-origin monitors (display left of / above the primary)

- **WHEN** the display arrangement places a monitor to the left of (or above) the primary,
  giving it a negative global origin — e.g. a real 3-monitor Mac observed as
  `monitor1 (-1728,0 1728x1117)`, `monitor2 (0,0 1920x1080 primary)`,
  `monitor3 (1920,0 1920x1080)`, so the virtual desktop spans x ∈ [-1728, 3840]
- **THEN** the global point space is signed and continuous across that whole range: a
  capture whose origin `(ox,oy)` is negative still maps image pixel `(px,py)` to global
  `(ox+px, oy+py)`, and `find --text` → `mouse click` round-trips on the negative-origin
  monitor with no special-casing. Monitor IDs are 1-indexed by the adapter's sort order and
  are NOT positional (id 1 may be the leftmost, negative-origin display).

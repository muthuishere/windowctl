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

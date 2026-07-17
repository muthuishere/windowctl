# find-text-ocr

## ADDED Requirements

### Requirement: On-screen text location via native OCR

`windowctl find --text <query>` SHALL locate visible on-screen text using the native
macOS Vision framework (`VNRecognizeTextRequest`) and return, for each match, a click
point in the same global point coordinate space that `windowctl mouse click` consumes.
It SHALL add no new Go dependency (pure CGO) and spawn no subprocess.

#### Scenario: Find text on the focused monitor

- **WHEN** `windowctl find --text "Submit"` is run with no scope flags
- **THEN** the focused monitor is captured and OCR'd
- **AND** every case-insensitive substring match is returned with `text`, `confidence`
  (0..1), `bounds` (global points), and `click` (global point at the match centroid)
- **AND** with `--json` the result is a JSON array ordered by descending confidence.

#### Scenario: Scope to a window or region

- **WHEN** `--title <s>`/`--app <s>` or `--x/--y/--w/--h` is given
- **THEN** only that window's rect (resolved like Move/Focus) or that region is captured
- **AND** returned click points are still absolute global points.

#### Scenario: Coordinate transform is correct on Retina

- **WHEN** a match is found in a capture whose origin is `(ox,oy)` on a HiDPI display
- **THEN** the returned click point equals `(ox + centroid_px, oy + centroid_py)` where
  the Vision normalized bottom-left bounding box has been flipped to top-left and scaled
  by the point-normalized image dimensions (1 image pixel == 1 global point)
- **AND** clicking that point activates the located UI element.

#### Scenario: No match

- **WHEN** the query matches no on-screen text
- **THEN** the command exits non-zero with `no on-screen text matched` (or `[]` with
  `--json`) — never a hardcoded/guessed coordinate.

#### Scenario: Screen Recording permission required

- **WHEN** `find` is run without the Screen Recording permission on macOS
- **THEN** it returns `ErrScreenCaptureDenied` pointing at `windowctl permissions --screen`.

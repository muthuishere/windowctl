# scroll-drag

## ADDED Requirements

### Requirement: Wheel scroll

`windowctl scroll --amount <n> [--horizontal] [--x <n> --y <n> [--monitor <m>]]` SHALL
emit wheel-scroll events. Positive amount scrolls down (or right with `--horizontal`);
negative reverses. If a point is given the cursor moves there first (monitor-relative rule
applies).

#### Scenario: Scroll at a point

- **WHEN** `windowctl scroll --amount 10 --x 500 --y 400`
- **THEN** the cursor moves to the global point and 10 lines of downward scroll are sent
  to whatever is under it.

### Requirement: Press-hold drag

`windowctl drag --from-x <n> --from-y <n> --to-x <n> --to-y <n> [--right|--middle]
[--monitor <m>]` SHALL press the button at the from-point, move through interpolated
points, and release at the to-point — usable for drag-drop, sliders, and text selection.

#### Scenario: Drag between two points

- **WHEN** a drag from `(200,200)` to `(600,400)` is requested
- **THEN** a button-down at the from-point, intermediate drag events, and a button-up at
  the to-point are emitted in order, so the target registers a continuous drag.

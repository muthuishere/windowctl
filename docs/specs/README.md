# windowctl — OpenSpec Specifications

**Source of truth:** [`docs/requirements.md`](../requirements.md) v0.1.

This folder slices the windowctl requirements into vertical, user-facing
capability specs in OpenSpec style. Each slice lives in its **own file**, is
self-contained (input → behavior → output), and references back to the
requirement IDs it covers.

Slices are numbered by **delivery value**, not by document order:

1. [`layout-apply`](./01-layout-apply.md) is the headline capability — the
   *why* of the tool.
2. [`os-spikes`](./02-os-spikes.md) is the tracer-bullet foundation that
   proves each target OS can actually drive a real window before any
   higher-level capability is built.

Everything else builds on those two.

---

## Status Index

| # | Spec | Status | Source requirements |
|---|------|--------|---------------------|
| 1 | [`layout-apply`](./01-layout-apply.md) | Planned | §4.4, FR-LAY-01, §11, OI-02 |
| 2 | [`os-spikes`](./02-os-spikes.md) | Partial | §7, §7.1, §10.3 |
| 3 | [`windows-list`](./03-windows-list.md) | Implemented | FR-WIN-01, FR-OUT-01, FR-OUT-02, §11 |
| 4 | [`monitors-list`](./04-monitors-list.md) | Implemented | FR-MON-01, FR-OUT-01, FR-OUT-02 |
| 5 | [`window-move-zone`](./05-window-move-zone.md) | Implemented | FR-MOV-01, FR-MOV-03, FR-MOV-04, §4.3.1, §4.3.2, §11 |
| 6 | [`window-move-coords`](./06-window-move-coords.md) | Implemented | FR-MOV-02, FR-MOV-04, §4.3.3, §11 |
| 7 | [`window-focus`](./07-window-focus.md) | Implemented | FR-FOC-01, §11 |
| 8 | [`go-library-api`](./08-go-library-api.md) | Partial | LIB-01, LIB-02, LIB-03, §8.3 |
| 9 | [`npm-distribution`](./09-npm-distribution.md) | Partial | §9.3, OI-05 |
| 10 | [`release-pipeline`](./10-release-pipeline.md) | Implemented | §9.1, §9.2 |
| 11 | [`ci-test-matrix`](./11-ci-test-matrix.md) | Implemented | §10.1, §10.2, §10.3, §10.4 |

**Status legend:** `Implemented` · `Partial` · `Planned` · `Not planned`.

---

## Architectural ground truth

From v1, the CLI has always been a thin wrapper over an internal library
package — every CLI command across these specs is implemented by delegating
to the library, not by reimplementing logic in the CLI layer. What was
**not** done initially was **exporting** that library as a public, importable
Go module. That export is a later capability and is captured in
[`go-library-api`](./08-go-library-api.md), whose status reflects this
split: the wrapper-shape and OS-mockable core have been in place from the
beginning, the public Go-module surface and dependency-isolation guarantee
came later.

Individual CLI specs do not restate "the CLI delegates to the library" in
every scenario — that is a global invariant covered once, here.

---

## Spec file conventions

Each slice file follows the same shape:

- **Front matter:** `Status`, `Source`, `Summary`.
- **`## Requirement: <name>`** — one per behavioral requirement in scope.
- **`### Scenario: <name>`** — Given/When/Then-style bullets under each
  requirement.

Status values match the index legend above. When a single spec mixes
implemented and planned requirements, per-requirement status is annotated
inline (e.g. *(Implemented from v1)* / *(Planned)*).

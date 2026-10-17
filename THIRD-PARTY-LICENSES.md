# Third-Party Licenses — windowctl

`windowctl` is distributed as a static Go binary, so any third-party code in the build is
redistributed with it and is attributed here.

windowctl itself is MIT — see `LICENSE`.

## Dependencies

| Dependency | Version | Licence | Linked into |
|---|---|---|---|
| `golang.org/x/sys` | v0.27.0 | BSD-3-Clause | **Windows builds only** |

That is the complete set — windowctl has exactly one third-party dependency.

`golang.org/x/sys` is used by the Windows adapter (`adapter_windows.go`) for the Win32
window APIs. It is **not** in the build graph for darwin or linux builds, both of which use
platform adapters written against system libraries directly. Verified with
`GOOS=windows go list -deps ./...` (present) against `GOOS=linux` and darwin (absent).

Copyright (c) 2009 The Go Authors. Licence text:
https://cs.opensource.google/go/x/sys/+/master:LICENSE

**No GPL / AGPL / LGPL / SSPL / MPL dependency exists in this project.**

## Regenerating

```sh
go-licenses report ./...                      # darwin/linux build graph
GOOS=windows go-licenses report ./...         # includes the Windows-only dependency
```

Note that a default `go-licenses` run on macOS or Linux will not show `golang.org/x/sys`,
because build constraints exclude the Windows adapter. Check the Windows graph explicitly.

# Vendored Swiss Ephemeris

C sources of the Swiss Ephemeris library, vendored verbatim for cgo compilation.

- Upstream: https://github.com/aloistr/swisseph
- Version: **2.10.03** (`SE_VERSION` in `sweph.h`)
- Commit: `59ac051b5a5812c684973ca0fcedb1c8c3e9c5dc` (2026-06-19)
- License: AGPL-3.0 (see `LICENSE` in this directory; dual-licensed by Astrodienst AG, we use the AGPL option — see repository `NOTICE`)

Vendored files: the library set only (`swecl.c swedate.c swehel.c swehouse.c swejpl.c swemmoon.c swemplan.c sweph.c swephlib.c` + headers). Upstream programs with `main()` (`swetest.c`, `swemini.c`, `swevents.c`, `swephgen4.c`, `obama.c`) and the ephe4 format support are intentionally excluded.

## Updating

1. Fetch upstream at the new commit (sparse checkout of `/*.c /*.h /LICENSE` is enough).
2. Copy the same file set over this directory; never patch vendored files locally.
3. Update version/commit in this README and re-run the golden test suite.

## Thread safety

The library keeps global state (open file handles, cached data) and is **not thread-safe**. `sweph.go` serializes all calls through a package mutex — do not bypass it.

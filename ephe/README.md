# Swiss Ephemeris data files

Planetary, lunar and asteroid ephemeris files, vendored verbatim from the
upstream repository. They cover the years 1800–2399, which fully contains
the API's accepted range (1801..2200).

- Upstream: https://github.com/aloistr/swisseph (`ephe/` directory)
- Commit: `59ac051b5a5812c684973ca0fcedb1c8c3e9c5dc` (same as `internal/sweph`)
- License: AGPL-3.0 (dual-licensed by Astrodienst AG, see repository `NOTICE`)

| File | Contents | SHA-256 |
|---|---|---|
| `sepl_18.se1` | planets 1800–2399 | `ca1393ceab3a44fbc895887cf789c68819ae6a1cbc9b22225872dbe4ccd99a66` |
| `semo_18.se1` | Moon 1800–2399 | `1ca07bd67c24374d77226180c20a4f9996cba013697894810518e7eb582ca4f7` |
| `seas_18.se1` | main asteroids incl. Chiron 1800–2399 | `a2cd8fc33807c78ca9a700c91c2e042258b12fc4796519e00781440b5ad8b2e2` |

The Docker image ships these files and sets `EPHE_PATH` to them, so the
container computes with the Swiss ephemeris (including Chiron) out of the
box. Without `EPHE_PATH` the service falls back to the built-in Moshier
approximation; both modes are exercised by the golden test suite.

## Updating

Fetch the same three files from the upstream commit you are vendoring,
update the table above (`shasum -a 256 ephe/*.se1`) and re-run the tests.

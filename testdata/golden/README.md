# Golden charts

Reference natal charts — the accuracy contract of this service. Each chart
is an input (`datetime_utc`, `lat`, `lon`, `house_system`) plus expected
planet longitudes and house cusps. CI runs every chart against both the
Swiss (bundled `ephe/` files) and the Moshier ephemeris on every commit;
any drift is a failing build.

Rules:

- 30 charts minimum (enforced by the test).
- Tolerances: planet longitudes ≤ 0.01°, house cusps ≤ 0.1°.
- Synthetic dates only — never real people's birth data (ПДн,
  docs/conventions.md).
- Coverage: USSR-era and modern Russia, polar latitudes (Porphyry
  fallback), southern hemisphere, equator, both range edges (1850, 2200),
  a leap day, every supported house system and the no-houses mode.

## Provenance

Values are generated with `swetest` — the Astrodienst reference utility,
same engine and data files as astro.com — built from the exact upstream
commit vendored in `internal/sweph`. Spot-checks of full charts (planets
and cusps) against the astro.com online swetest returned byte-identical
values.

## Regenerating

1. Build the reference utility from the vendored commit (kept out of the
   repository on purpose — only the library set is vendored):

   ```sh
   curl -fsSLO https://raw.githubusercontent.com/aloistr/swisseph/<vendored commit>/swetest.c
   cc -O2 -o swetest swetest.c internal/sweph/*.c -Iinternal/sweph -lm
   ```

2. Regenerate (the chart list lives in `gen/main.go`):

   ```sh
   go run ./testdata/golden/gen -swetest ./swetest -ephe ephe -out testdata/golden
   ```

3. Re-run `go test ./internal/chart` and eyeball the diff: expected values
   may only change when the vendored library or data files change.

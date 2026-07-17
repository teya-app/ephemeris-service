# Teya Ephemeris Service

An HTTP sidecar service that wraps the [Swiss Ephemeris](https://github.com/aloistr/swisseph) library and turns birth data into precise astronomical positions. It is the calculation core of **Teya**: date and place in — degrees out, nothing else.

## Why this is a separate public repository

Swiss Ephemeris is dual-licensed: AGPL-3.0 or a commercial license from Astrodienst AG. Teya uses it under **AGPL-3.0**. Section 13 of the AGPL requires offering the complete source of the network service that incorporates the library to its users — this repository is that source.

The isolation rules we hold ourselves to:

1. Swiss Ephemeris is linked (via cgo) **only** inside this service — its own process/container.
2. The rest of the Teya backend talks to this service over HTTP with plain JSON and contains no Swiss Ephemeris code.
3. Any change to this service is published here **before** it is deployed to production.
4. The product exposes a permanent "Source" link pointing to this repository.

## API

`POST /v1/chart`

```jsonc
// request
{
  "datetime_utc": "1990-05-17T21:15:00Z",   // RFC3339, years 1801..2200
                                             // (offsets accepted, converted to UT)
  "lat": 59.9386,                            // -90..90
  "lon": 30.3141,                            // -180..180
  "house_system": "placidus"                 // placidus | koch | whole_sign |
                                             // equal | porphyry | none
                                             // (default placidus; "none" =
                                             // unknown birth time, no houses)
}

// response (real output for the request above, lists abbreviated)
{
  "planets": [
    { "name": "sun", "lon": 56.6952, "sign": "taurus", "sign_lon": 26.6952,
      "house": 4, "speed": 0.9634, "retrograde": false }
    // sun..pluto, mean_node, lilith; chiron when ephemeris files are available.
    // "house" is omitted with house_system=none.
  ],
  "houses":  [ { "num": 1, "cusp_lon": 266.3371, "sign": "sagittarius" } ],
  "angles":  { "asc": 266.3371, "asc_sign": "sagittarius", "mc": 226.838, "mc_sign": "scorpio" },
  "aspects": [ { "p1": "sun", "p2": "moon", "type": "square", "orb": 0.7594 } ],
  "meta": {
    "engine_version": "2.10.03",
    "ephemeris": "moshier",        // or "swiss" when EPHE_PATH is set
    "house_system": "placidus",
    "polar_fallback": false        // always present; true: quadrant system
                                   // undefined at this latitude, cusps
                                   // computed with Porphyry
  }
}
```

With `house_system: "none"` the response has no `houses` and no `angles`, and planets carry no `house` field. Errors are always `{"error": "..."}` with 4xx/5xx and never echo the submitted values.

`GET /healthz` → `{ "status": "ok", "engine_version": "...", "ephemeris": "..." }`

The contract is intentionally dumb: no user identifiers, no persistence, no business logic. The consuming application never computes positions on its own (and never lets an LLM guess them).

## Running

```sh
go run ./cmd/ephemeris-service          # listens on :8080, Moshier ephemeris
# or
docker build -t ephemeris-service . && docker run -p 8080:8080 ephemeris-service
```

Configuration (environment only): `ADDR` (default `:8080`), `EPHE_PATH` (directory with Swiss Ephemeris `.se1` files; unset = built-in Moshier approximation, ~0.1″ precision for planets, no Chiron), `LOG_LEVEL` (`debug|info|warn|error`).

The Docker image bundles the `.se1` files for 1800–2399 ([`ephe/`](ephe/), ~2 MB) and sets `EPHE_PATH` to them, so containers compute with the Swiss ephemeris — Chiron included — out of the box. `go run` without `EPHE_PATH` uses Moshier.

When `EPHE_PATH` is set, the files are probed at startup and the service refuses to start if they are unusable — the library would otherwise fall back to Moshier silently while the service reports Swiss precision.

`ephemeris-service healthcheck` probes `/healthz` of the local server and exits 0/1 — used by the Docker `HEALTHCHECK` (the slim runtime image has no curl/wget).

Request bodies contain birth data (personal data) and are never logged.

## Accuracy

The golden test suite ([`testdata/golden/`](testdata/golden/)) holds 30 reference charts generated with the Astrodienst reference utility `swetest` and spot-checked byte-identical against astro.com:

- planet longitudes within **0.01°**
- house cusps within **0.1°**
- covers USSR-era and modern Russian dates, polar latitudes, the southern hemisphere, both range edges and every supported house system

CI runs every chart in both Swiss and Moshier modes on every commit and fails on any drift.

## Status

v0 implemented: chart computation (planets, houses, angles, aspects) over HTTP; Swiss ephemeris files bundled in the image, Moshier fallback without them; golden suite of 30 charts in CI. Historical timezone handling (local birth time → UTC) lives in the consuming application.

## Security

Please report vulnerabilities via [GitHub private vulnerability reporting](../../security/advisories/new) rather than public issues.

## License

[AGPL-3.0](LICENSE). Swiss Ephemeris is © Astrodienst AG, Zurich — dual-licensed AGPL-3.0 / commercial; see [NOTICE](NOTICE).

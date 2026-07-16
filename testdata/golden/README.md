# Golden charts

Reference natal charts used as the accuracy contract of this service. Each chart is an input (`datetime_utc`, `lat`, `lon`, `house_system`) plus expected output positions, cross-checked against astro.com.

Rules:

- 30 charts minimum before the first production release.
- Tolerances: planet longitudes ≤ 0.01°, house cusps ≤ 0.1°.
- Must include births from the USSR era (decree time, DST transitions) and the 1991–2014 Russian timezone turbulence — historical timezone handling is the classic failure mode of astrological software.
- CI runs every chart on every commit; any drift is a failing build.

Populated together with the first service code.

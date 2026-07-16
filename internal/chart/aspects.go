package chart

import "math"

type aspectDef struct {
	name  string
	angle float64
	orb   float64
}

// Major (Ptolemaic) aspects with MVP orbs. Orb policy may become
// configurable later; keep it in one place.
var aspectDefs = []aspectDef{
	{"conjunction", 0, 8},
	{"sextile", 60, 6},
	{"square", 90, 8},
	{"trine", 120, 8},
	{"opposition", 180, 8},
}

// aspectBodies are the planets participating in aspect calculation.
// Points (nodes, lilith) are excluded to keep the result focused; the
// consuming product can request them explicitly once needed.
var aspectBodies = map[string]bool{
	"sun": true, "moon": true, "mercury": true, "venus": true, "mars": true,
	"jupiter": true, "saturn": true, "uranus": true, "neptune": true,
	"pluto": true, "chiron": true,
}

// angularSep returns the smallest angle between two longitudes, [0, 180].
func angularSep(a, b float64) float64 {
	d := math.Abs(normalizeLon(a) - normalizeLon(b))
	if d > 180 {
		d = 360 - d
	}
	return d
}

// computeAspects finds major aspects between all pairs of aspect-bearing
// planets. Planets must be in a stable order; output order follows input.
func computeAspects(planets []Planet) []Aspect {
	aspects := []Aspect{}
	for i := 0; i < len(planets); i++ {
		if !aspectBodies[planets[i].Name] {
			continue
		}
		for j := i + 1; j < len(planets); j++ {
			if !aspectBodies[planets[j].Name] {
				continue
			}
			sep := angularSep(planets[i].Lon, planets[j].Lon)
			for _, def := range aspectDefs {
				orb := math.Abs(sep - def.angle)
				if orb <= def.orb {
					aspects = append(aspects, Aspect{
						P1:   planets[i].Name,
						P2:   planets[j].Name,
						Type: def.name,
						Orb:  round4(orb),
					})
					break // a pair matches at most one major aspect
				}
			}
		}
	}
	return aspects
}

// round4 rounds to 4 decimal places — enough for 0.36" precision,
// keeps JSON payloads tidy.
func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

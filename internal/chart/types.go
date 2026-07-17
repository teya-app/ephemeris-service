// Package chart turns raw ephemeris output into a natal chart:
// planets in signs and houses, chart angles and aspects.
package chart

import "time"

// Input is a fully validated chart request.
type Input struct {
	DatetimeUTC time.Time
	Lat         float64
	Lon         float64
	// HouseSystem is one of HouseSystems keys, or "none" to skip houses
	// and angles (birth time unknown).
	HouseSystem string
}

// Planet is a celestial body positioned on the ecliptic.
type Planet struct {
	Name       string  `json:"name"`
	Lon        float64 `json:"lon"`      // ecliptic longitude, degrees [0, 360)
	Sign       string  `json:"sign"`     // zodiac sign, lowercase english
	SignLon    float64 `json:"sign_lon"` // degrees within the sign [0, 30)
	House      int     `json:"house,omitempty"`
	Speed      float64 `json:"speed"` // degrees/day
	Retrograde bool    `json:"retrograde"`
}

// House is a single house cusp.
type House struct {
	Num     int     `json:"num"` // 1..12
	CuspLon float64 `json:"cusp_lon"`
	Sign    string  `json:"sign"`
}

// Angles are the main chart angles.
type Angles struct {
	Asc     float64 `json:"asc"`
	AscSign string  `json:"asc_sign"`
	MC      float64 `json:"mc"`
	MCSign  string  `json:"mc_sign"`
}

// Aspect is an angular relation between two planets.
type Aspect struct {
	P1   string  `json:"p1"`
	P2   string  `json:"p2"`
	Type string  `json:"type"`
	Orb  float64 `json:"orb"` // deviation from the exact angle, degrees
}

// Meta describes how the chart was computed.
type Meta struct {
	EngineVersion string `json:"engine_version"`
	Ephemeris     string `json:"ephemeris"` // "swiss" | "moshier"
	HouseSystem   string `json:"house_system"`
	// PolarFallback is true when a quadrant house system was undefined at
	// this latitude and cusps were computed with Porphyry instead.
	PolarFallback bool `json:"polar_fallback"`
}

// Chart is the complete calculation result.
type Chart struct {
	Planets []Planet `json:"planets"`
	Houses  []House  `json:"houses,omitempty"`
	Angles  *Angles  `json:"angles,omitempty"`
	Aspects []Aspect `json:"aspects"`
	Meta    Meta     `json:"meta"`
}

// HouseSystems maps API names to Swiss Ephemeris house system codes.
var HouseSystems = map[string]byte{
	"placidus":   'P',
	"koch":       'K',
	"whole_sign": 'W',
	"equal":      'E',
	"porphyry":   'O',
}

// HouseSystemNone disables house calculation (unknown birth time).
const HouseSystemNone = "none"

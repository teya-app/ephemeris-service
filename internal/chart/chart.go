package chart

import (
	"fmt"
	"log/slog"

	"github.com/teya-app/ephemeris-service/internal/sweph"
)

// computedBodies is the ordered set of bodies included in every chart.
// Chiron is attempted last and skipped when the ephemeris cannot provide
// it (the built-in Moshier approximation has no asteroid data).
var computedBodies = []struct {
	body sweph.Body
	name string
}{
	{sweph.Sun, "sun"},
	{sweph.Moon, "moon"},
	{sweph.Mercury, "mercury"},
	{sweph.Venus, "venus"},
	{sweph.Mars, "mars"},
	{sweph.Jupiter, "jupiter"},
	{sweph.Saturn, "saturn"},
	{sweph.Uranus, "uranus"},
	{sweph.Neptune, "neptune"},
	{sweph.Pluto, "pluto"},
	{sweph.MeanNode, "mean_node"},
	{sweph.MeanApogee, "lilith"},
	{sweph.Chiron, "chiron"},
}

// optionalBodies may legitimately fail depending on available ephemeris data.
var optionalBodies = map[string]bool{"chiron": true}

// Engine computes natal charts. Safe for concurrent use: the underlying
// sweph package serializes library access.
type Engine struct {
	useSwiss bool
	log      *slog.Logger
}

// NewEngine creates an Engine. If ephePath is non-empty, it must point to a
// directory with Swiss Ephemeris .se1 data files; otherwise the built-in
// Moshier approximation is used (precise enough for natal work: ~0.1″ for
// planets, but no Chiron).
func NewEngine(ephePath string, log *slog.Logger) *Engine {
	if ephePath != "" {
		sweph.SetEphePath(ephePath)
	}
	return &Engine{useSwiss: ephePath != "", log: log}
}

// Ephemeris returns the active ephemeris kind: "swiss" or "moshier".
func (e *Engine) Ephemeris() string {
	if e.useSwiss {
		return "swiss"
	}
	return "moshier"
}

// EngineVersion returns the Swiss Ephemeris library version.
func (e *Engine) EngineVersion() string {
	return sweph.Version()
}

// Compute builds a full natal chart for a validated input.
func (e *Engine) Compute(in Input) (*Chart, error) {
	jd := sweph.JulDayUT(in.DatetimeUTC)

	planets := make([]Planet, 0, len(computedBodies))
	for _, b := range computedBodies {
		res, err := sweph.CalcUT(jd, b.body, e.useSwiss)
		if err != nil {
			if optionalBodies[b.name] {
				e.log.Debug("optional body skipped", "body", b.name, "reason", err.Error())
				continue
			}
			return nil, fmt.Errorf("calc %s: %w", b.name, err)
		}
		lon := normalizeLon(res.Lon)
		planets = append(planets, Planet{
			Name:       b.name,
			Lon:        round4(lon),
			Sign:       signFor(lon),
			SignLon:    round4(signLon(lon)),
			Speed:      round4(res.LonSpeed),
			Retrograde: res.LonSpeed < 0,
		})
	}

	c := &Chart{
		Planets: planets,
		Aspects: computeAspects(planets),
		Meta: Meta{
			EngineVersion: sweph.Version(),
			Ephemeris:     e.Ephemeris(),
			HouseSystem:   in.HouseSystem,
		},
	}

	if in.HouseSystem != HouseSystemNone {
		hsys, ok := HouseSystems[in.HouseSystem]
		if !ok {
			return nil, fmt.Errorf("unknown house system %q", in.HouseSystem)
		}
		hr := sweph.HousesUT(jd, in.Lat, in.Lon, hsys)
		c.Meta.PolarFallback = hr.PolarFallback

		houses := make([]House, 12)
		for i := 1; i <= 12; i++ {
			cusp := normalizeLon(hr.Cusps[i])
			houses[i-1] = House{Num: i, CuspLon: round4(cusp), Sign: signFor(cusp)}
		}
		c.Houses = houses
		c.Angles = &Angles{
			Asc:     round4(normalizeLon(hr.Asc)),
			AscSign: signFor(hr.Asc),
			MC:      round4(normalizeLon(hr.MC)),
			MCSign:  signFor(hr.MC),
		}
		for i := range c.Planets {
			c.Planets[i].House = houseFor(c.Planets[i].Lon, hr.Cusps)
		}
	}

	return c, nil
}

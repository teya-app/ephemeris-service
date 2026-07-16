package chart

import (
	"log/slog"
	"testing"
	"time"
)

func testEngine() *Engine {
	return NewEngine("", slog.New(slog.DiscardHandler))
}

// TestComputeSmoke checks structural invariants and coarse astronomical
// facts that hold regardless of ephemeris precision. Exact positions are
// verified by the golden test suite against astro.com references.
func TestComputeSmoke(t *testing.T) {
	c, err := testEngine().Compute(Input{
		DatetimeUTC: time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC),
		Lat:         55.7558, // Moscow
		Lon:         37.6173,
		HouseSystem: "placidus",
	})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	byName := map[string]Planet{}
	for _, p := range c.Planets {
		byName[p.Name] = p
	}

	// Moshier has no Chiron; everything else must be present.
	required := []string{
		"sun", "moon", "mercury", "venus", "mars", "jupiter",
		"saturn", "uranus", "neptune", "pluto", "mean_node", "lilith",
	}
	for _, name := range required {
		if _, ok := byName[name]; !ok {
			t.Errorf("missing planet %q", name)
		}
	}

	// On 2000-01-01 the Sun is ~10° into Capricorn (entered ~Dec 22,
	// moving ~1.02°/day from 270°).
	sun := byName["sun"]
	if sun.Sign != "capricorn" {
		t.Errorf("sun sign = %q, want capricorn", sun.Sign)
	}
	if sun.Lon < 279 || sun.Lon > 282 {
		t.Errorf("sun lon = %v, want within [279, 282]", sun.Lon)
	}
	if sun.Retrograde {
		t.Error("sun cannot be retrograde")
	}

	// The Moon moves 11.7..15.4°/day and is never retrograde.
	moon := byName["moon"]
	if moon.Speed < 11 || moon.Speed > 16 {
		t.Errorf("moon speed = %v °/day, want within [11, 16]", moon.Speed)
	}

	for _, p := range c.Planets {
		if p.Lon < 0 || p.Lon >= 360 {
			t.Errorf("%s lon = %v, out of [0, 360)", p.Name, p.Lon)
		}
		if p.Retrograde != (p.Speed < 0) {
			t.Errorf("%s: retrograde=%v inconsistent with speed=%v", p.Name, p.Retrograde, p.Speed)
		}
		if p.House < 1 || p.House > 12 {
			t.Errorf("%s house = %d, want 1..12", p.Name, p.House)
		}
	}

	if len(c.Houses) != 12 {
		t.Fatalf("houses = %d, want 12", len(c.Houses))
	}
	if c.Angles == nil {
		t.Fatal("angles missing")
	}
	// For quadrant systems the 1st cusp is the Ascendant.
	if diff := c.Angles.Asc - c.Houses[0].CuspLon; diff > 0.01 || diff < -0.01 {
		t.Errorf("asc=%v differs from cusp1=%v", c.Angles.Asc, c.Houses[0].CuspLon)
	}
	if c.Meta.Ephemeris != "moshier" {
		t.Errorf("ephemeris = %q, want moshier", c.Meta.Ephemeris)
	}
	if c.Meta.EngineVersion == "" {
		t.Error("engine version empty")
	}
}

// TestComputeNoHouses covers the unknown-birth-time mode.
func TestComputeNoHouses(t *testing.T) {
	c, err := testEngine().Compute(Input{
		DatetimeUTC: time.Date(1985, 7, 13, 0, 0, 0, 0, time.UTC),
		Lat:         0,
		Lon:         0,
		HouseSystem: HouseSystemNone,
	})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(c.Houses) != 0 || c.Angles != nil {
		t.Error("houses/angles must be absent when house_system=none")
	}
	for _, p := range c.Planets {
		if p.House != 0 {
			t.Errorf("%s has house %d in no-houses mode", p.Name, p.House)
		}
	}
}

// TestComputeDeterminism: same input → same output.
func TestComputeDeterminism(t *testing.T) {
	in := Input{
		DatetimeUTC: time.Date(1969, 11, 20, 6, 30, 0, 0, time.UTC),
		Lat:         59.9386,
		Lon:         30.3141,
		HouseSystem: "placidus",
	}
	e := testEngine()
	a, err := e.Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	b, err := e.Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	for i := range a.Planets {
		if a.Planets[i] != b.Planets[i] {
			t.Errorf("non-deterministic result for %s", a.Planets[i].Name)
		}
	}
}

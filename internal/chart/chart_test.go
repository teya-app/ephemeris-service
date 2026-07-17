package chart

import (
	"log/slog"
	"math"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine("", slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return e
}

// TestComputeSmoke checks structural invariants and coarse astronomical
// facts that hold regardless of ephemeris precision. Exact positions are
// verified by the golden test suite against astro.com references.
func TestComputeSmoke(t *testing.T) {
	c, err := testEngine(t).Compute(Input{
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

	required := []string{
		"sun", "moon", "mercury", "venus", "mars", "jupiter",
		"saturn", "uranus", "neptune", "pluto", "mean_node", "lilith",
	}
	for _, name := range required {
		if _, ok := byName[name]; !ok {
			t.Errorf("missing planet %q", name)
		}
	}

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

func TestComputeNoHouses(t *testing.T) {
	c, err := testEngine(t).Compute(Input{
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

func TestComputeDeterminism(t *testing.T) {
	in := Input{
		DatetimeUTC: time.Date(1969, 11, 20, 6, 30, 0, 0, time.UTC),
		Lat:         59.9386,
		Lon:         30.3141,
		HouseSystem: "placidus",
	}
	e := testEngine(t)
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

func TestNewEngineRejectsUnusableEphePath(t *testing.T) {
	if _, err := NewEngine(t.TempDir(), slog.New(slog.DiscardHandler)); err == nil {
		t.Fatal("NewEngine with an empty ephemeris dir must fail")
	}
}

func TestComputePolarFallback(t *testing.T) {
	for _, tt := range []struct {
		name string
		lat  float64
	}{
		{"murmansk", 68.97},
		{"north pole", 90},
	} {
		t.Run(tt.name, func(t *testing.T) {
			c, err := testEngine(t).Compute(Input{
				DatetimeUTC: time.Date(1990, 5, 17, 21, 15, 0, 0, time.UTC),
				Lat:         tt.lat,
				Lon:         33.08,
				HouseSystem: "placidus",
			})
			if err != nil {
				t.Fatalf("Compute: %v", err)
			}
			if !c.Meta.PolarFallback {
				t.Error("polar_fallback must be true beyond the polar circle")
			}
			if len(c.Houses) != 12 {
				t.Fatalf("houses = %d, want 12", len(c.Houses))
			}
			for _, h := range c.Houses {
				if h.CuspLon < 0 || h.CuspLon >= 360 {
					t.Errorf("house %d cusp %v out of [0, 360)", h.Num, h.CuspLon)
				}
			}
			for _, p := range c.Planets {
				if p.House < 1 || p.House > 12 {
					t.Errorf("%s house = %d, want 1..12", p.Name, p.House)
				}
			}
		})
	}
}

func TestComputeHouseSystems(t *testing.T) {
	in := Input{
		DatetimeUTC: time.Date(1990, 5, 17, 21, 15, 0, 0, time.UTC),
		Lat:         59.9386,
		Lon:         30.3141,
	}
	e := testEngine(t)
	for name := range HouseSystems {
		t.Run(name, func(t *testing.T) {
			in := in
			in.HouseSystem = name
			c, err := e.Compute(in)
			if err != nil {
				t.Fatalf("Compute: %v", err)
			}
			if len(c.Houses) != 12 || c.Angles == nil {
				t.Fatalf("incomplete chart: %d houses, angles=%v", len(c.Houses), c.Angles)
			}
			if c.Meta.PolarFallback {
				t.Errorf("unexpected polar fallback at lat %v", in.Lat)
			}
			switch name {
			case "placidus", "koch", "porphyry":
				if d := lonDiff(c.Angles.Asc, c.Houses[0].CuspLon); d > 0.01 {
					t.Errorf("asc=%v differs from cusp1=%v", c.Angles.Asc, c.Houses[0].CuspLon)
				}
			case "whole_sign":
				for _, h := range c.Houses {
					if r := math.Mod(h.CuspLon, 30); r > 1e-9 && 30-r > 1e-9 {
						t.Errorf("whole-sign cusp %d = %v, not a multiple of 30°", h.Num, h.CuspLon)
					}
				}
			case "equal":
				for i := 1; i < 12; i++ {
					step := normalizeLon(c.Houses[i].CuspLon - c.Houses[i-1].CuspLon)
					if step < 29.99 || step > 30.01 {
						t.Errorf("equal-house step %d→%d = %v, want 30°", i, i+1, step)
					}
				}
			}
		})
	}
}

func TestComputeConcurrent(t *testing.T) {
	in := Input{
		DatetimeUTC: time.Date(1969, 11, 20, 6, 30, 0, 0, time.UTC),
		Lat:         59.9386,
		Lon:         30.3141,
		HouseSystem: "placidus",
	}
	e := testEngine(t)
	want, err := e.Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := e.Compute(in)
			if err != nil {
				t.Errorf("Compute: %v", err)
				return
			}
			for i := range want.Planets {
				if got.Planets[i] != want.Planets[i] {
					t.Errorf("concurrent result differs for %s", want.Planets[i].Name)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func swissEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := NewEngine(filepath.Join("..", "..", "ephe"), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("NewEngine(ephe): %v", err)
	}
	return e
}

func TestSwissEphemeris(t *testing.T) {
	e := swissEngine(t)
	if e.Ephemeris() != "swiss" {
		t.Fatalf("ephemeris = %q, want swiss", e.Ephemeris())
	}

	in := Input{
		DatetimeUTC: time.Date(1990, 5, 17, 21, 15, 0, 0, time.UTC),
		Lat:         59.9386,
		Lon:         30.3141,
		HouseSystem: "placidus",
	}
	c, err := e.Compute(in)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if len(c.Planets) != 13 {
		t.Errorf("planets = %d, want 13 (chiron included)", len(c.Planets))
	}
	var chiron Planet
	for _, p := range c.Planets {
		if p.Name == "chiron" {
			chiron = p
		}
	}
	if chiron.Name == "" {
		t.Fatal("chiron missing with swiss files")
	}
	if chiron.Sign != "cancer" {
		t.Errorf("chiron sign = %q, want cancer (1990-05-17)", chiron.Sign)
	}

	// Moshier and Swiss must agree far below the golden tolerance.
	m, err := testEngine(t).Compute(in)
	if err != nil {
		t.Fatalf("Compute (moshier): %v", err)
	}
	moshier := map[string]float64{}
	for _, p := range m.Planets {
		moshier[p.Name] = p.Lon
	}
	for _, p := range c.Planets {
		if p.Name == "chiron" {
			continue
		}
		if d := lonDiff(p.Lon, moshier[p.Name]); d > 0.01 {
			t.Errorf("%s: swiss %v vs moshier %v (Δ%.4f°)", p.Name, p.Lon, moshier[p.Name], d)
		}
	}
}

func TestSwissEphemerisCoversAPIRange(t *testing.T) {
	e := swissEngine(t)
	for _, dt := range []time.Time{
		time.Date(1801, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2200, 12, 31, 23, 59, 59, 0, time.UTC),
	} {
		if _, err := e.Compute(Input{DatetimeUTC: dt, Lat: 0, Lon: 0, HouseSystem: "placidus"}); err != nil {
			t.Errorf("Compute(%s): %v", dt.Format(time.RFC3339), err)
		}
	}
}

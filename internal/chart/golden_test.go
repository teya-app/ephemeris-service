package chart

import (
	"encoding/json"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// goldenChart is one reference chart; values are generated with the
// swetest reference utility and spot-checked against astro.com.
// See testdata/golden/README.md for the rules and the procedure.
type goldenChart struct {
	Name  string `json:"name"`
	Input struct {
		DatetimeUTC string  `json:"datetime_utc"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		HouseSystem string  `json:"house_system"`
	} `json:"input"`
	Tolerance struct {
		PlanetDeg float64 `json:"planet_deg"` // default 0.01
		CuspDeg   float64 `json:"cusp_deg"`   // default 0.1
	} `json:"tolerance"`
	Expected struct {
		Planets []struct {
			Name string  `json:"name"`
			Lon  float64 `json:"lon"`
		} `json:"planets"`
		Houses []struct {
			Num     int     `json:"num"`
			CuspLon float64 `json:"cusp_lon"`
		} `json:"houses"`
	} `json:"expected"`
}

func TestGoldenCharts(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "testdata", "golden", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 30 {
		t.Fatalf("golden set must hold at least 30 charts, found %d", len(files))
	}

	// Both modes must satisfy the golden contract.
	for _, mode := range []struct {
		name     string
		ephePath string
	}{
		{"swiss", filepath.Join("..", "..", "ephe")},
		{"moshier", ""},
	} {
		t.Run(mode.name, func(t *testing.T) {
			engine, err := NewEngine(mode.ephePath, slog.New(slog.DiscardHandler))
			if err != nil {
				t.Fatalf("NewEngine: %v", err)
			}
			for _, file := range files {
				t.Run(filepath.Base(file), func(t *testing.T) {
					runGoldenChart(t, engine, file)
				})
			}
		})
	}
}

func runGoldenChart(t *testing.T, engine *Engine, file string) {
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var g goldenChart
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatalf("bad golden file: %v", err)
	}
	if g.Tolerance.PlanetDeg == 0 {
		g.Tolerance.PlanetDeg = 0.01
	}
	if g.Tolerance.CuspDeg == 0 {
		g.Tolerance.CuspDeg = 0.1
	}

	dt, err := time.Parse(time.RFC3339, g.Input.DatetimeUTC)
	if err != nil {
		t.Fatalf("bad datetime: %v", err)
	}
	hsys := g.Input.HouseSystem
	if hsys == "" {
		hsys = "placidus"
	}

	c, err := engine.Compute(Input{
		DatetimeUTC: dt, Lat: g.Input.Lat, Lon: g.Input.Lon, HouseSystem: hsys,
	})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	planets := map[string]float64{}
	for _, p := range c.Planets {
		planets[p.Name] = p.Lon
	}
	for _, want := range g.Expected.Planets {
		got, ok := planets[want.Name]
		if !ok {
			// Moshier has no asteroid data; optional bodies are absent.
			if optionalBodies[want.Name] && engine.Ephemeris() == "moshier" {
				continue
			}
			t.Errorf("planet %q missing", want.Name)
			continue
		}
		if d := lonDiff(got, want.Lon); d > g.Tolerance.PlanetDeg {
			t.Errorf("%s: lon %v, want %v (Δ%.4f° > %v°)", want.Name, got, want.Lon, d, g.Tolerance.PlanetDeg)
		}
	}

	cusps := map[int]float64{}
	for _, h := range c.Houses {
		cusps[h.Num] = h.CuspLon
	}
	for _, want := range g.Expected.Houses {
		got, ok := cusps[want.Num]
		if !ok {
			t.Errorf("house %d missing", want.Num)
			continue
		}
		if d := lonDiff(got, want.CuspLon); d > g.Tolerance.CuspDeg {
			t.Errorf("house %d: cusp %v, want %v (Δ%.4f° > %v°)", want.Num, got, want.CuspLon, d, g.Tolerance.CuspDeg)
		}
	}
	if hsys == HouseSystemNone && len(c.Houses) != 0 {
		t.Errorf("houses must be absent with house_system=none, got %d", len(c.Houses))
	}
}

// lonDiff is the circular difference between two longitudes, degrees.
func lonDiff(a, b float64) float64 {
	d := math.Abs(normalizeLon(a) - normalizeLon(b))
	if d > 180 {
		d = 360 - d
	}
	return d
}

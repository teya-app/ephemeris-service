// Command gen (re)generates the golden chart files from the reference
// swetest utility. It is excluded from normal builds by living under
// testdata; see testdata/golden/README.md for the full procedure.
//
// Usage:
//
//	go run ./testdata/golden/gen -swetest /path/to/swetest [-ephe ephe] [-out testdata/golden]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/teya-app/ephemeris-service/internal/chart"
)

type goldenInput struct {
	DatetimeUTC string  `json:"datetime_utc"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	HouseSystem string  `json:"house_system"`
}

type goldenPlanet struct {
	Name string  `json:"name"`
	Lon  float64 `json:"lon"`
}

type goldenHouse struct {
	Num     int     `json:"num"`
	CuspLon float64 `json:"cusp_lon"`
}

type goldenFile struct {
	Name     string      `json:"name"`
	Source   string      `json:"source"`
	Input    goldenInput `json:"input"`
	Expected struct {
		Planets []goldenPlanet `json:"planets"`
		Houses  []goldenHouse  `json:"houses,omitempty"`
	} `json:"expected"`
}

// charts is the full golden set. Synthetic inputs only — no real people's
// birth data (docs/conventions.md).
var charts = []struct {
	slug        string
	datetimeUTC string
	lat, lon    float64
	houseSystem string
}{
	// USSR and Russia across the decades (the core audience geography).
	{"ussr-moscow-1948", "1948-03-11T05:30:00Z", 55.7558, 37.6173, "placidus"},
	{"ussr-leningrad-1961", "1961-04-12T09:07:00Z", 59.9386, 30.3141, "placidus"},
	{"ussr-sverdlovsk-1972", "1972-08-24T17:45:00Z", 56.8389, 60.6057, "placidus"},
	{"ussr-novosibirsk-1985", "1985-01-05T02:20:00Z", 55.0084, 82.9357, "placidus"},
	{"russia-moscow-1991", "1991-12-26T10:00:00Z", 55.7558, 37.6173, "placidus"},
	{"russia-samara-1994", "1994-06-30T23:59:00Z", 53.1959, 50.1002, "koch"},
	{"russia-kaliningrad-leap-2000", "2000-02-29T12:00:00Z", 54.7104, 20.4522, "placidus"},
	{"russia-vladivostok-2010", "2010-11-07T21:30:00Z", 43.1155, 131.8855, "placidus"},
	{"russia-moscow-2020", "2020-03-29T00:00:00Z", 55.7558, 37.6173, "whole_sign"},
	{"russia-kazan-2023", "2023-07-01T15:42:00Z", 55.8304, 49.0661, "equal"},
	// Beyond the polar circle: quadrant systems fall back to Porphyry.
	{"polar-murmansk-1990", "1990-05-17T21:15:00Z", 68.9585, 33.0827, "placidus"},
	{"polar-norilsk-solstice-1975", "1975-12-21T15:00:00Z", 69.3558, 88.1893, "placidus"},
	{"polar-longyearbyen-2005", "2005-06-21T12:00:00Z", 78.2232, 15.6267, "koch"},
	// High latitude but below the circle: Placidus still defined.
	{"north-reykjavik-1980", "1980-06-19T23:50:00Z", 64.1466, -21.9426, "placidus"},
	// Southern hemisphere.
	{"south-sydney-1988", "1988-09-14T03:25:00Z", -33.8688, 151.2093, "placidus"},
	{"south-buenos-aires-1979", "1979-07-02T14:10:00Z", -34.6037, -58.3816, "placidus"},
	{"south-cape-town-1995", "1995-10-23T08:45:00Z", -33.9249, 18.4241, "equal"},
	{"south-wellington-2012", "2012-12-21T11:11:00Z", -41.2865, 174.7762, "placidus"},
	// Equator and tropics.
	{"equator-quito-equinox-1983", "1983-03-21T18:00:00Z", -0.1807, -78.4678, "placidus"},
	{"tropic-singapore-2001", "2001-09-09T09:09:00Z", 1.3521, 103.8198, "whole_sign"},
	{"tropic-honolulu-1959", "1959-08-21T10:30:00Z", 21.3069, -157.8583, "equal"},
	// Western hemisphere.
	{"west-new-york-1969", "1969-07-20T20:17:00Z", 40.7128, -74.006, "placidus"},
	{"west-los-angeles-1994", "1994-01-17T12:30:00Z", 34.0522, -118.2437, "porphyry"},
	{"west-mexico-city-1990", "1990-06-10T04:44:00Z", 19.4326, -99.1332, "placidus"},
	// Historic range edges and old dates.
	{"edge-london-1850", "1850-05-01T10:00:00Z", 51.5074, -0.1278, "placidus"},
	{"edge-paris-1900", "1900-01-01T00:00:00Z", 48.8566, 2.3522, "placidus"},
	{"edge-tokyo-1923", "1923-09-01T02:58:00Z", 35.6762, 139.6503, "placidus"},
	// Future dates: transits and solar returns.
	{"future-moscow-2100", "2100-01-01T12:00:00Z", 55.7558, 37.6173, "placidus"},
	{"future-spb-2200", "2200-06-15T00:00:00Z", 59.9386, 30.3141, "placidus"},
	// Unknown birth time: planets only.
	{"no-houses-1987", "1987-04-15T06:30:00Z", 0, 0, "none"},
}

// planetNames maps swetest object names to the service's API names.
var planetNames = map[string]string{
	"Sun":         "sun",
	"Moon":        "moon",
	"Mercury":     "mercury",
	"Venus":       "venus",
	"Mars":        "mars",
	"Jupiter":     "jupiter",
	"Saturn":      "saturn",
	"Uranus":      "uranus",
	"Neptune":     "neptune",
	"Pluto":       "pluto",
	"mean Node":   "mean_node",
	"mean Apogee": "lilith",
	"Chiron":      "chiron",
}

func main() {
	swetest := flag.String("swetest", "", "path to the swetest binary (required)")
	ephe := flag.String("ephe", "ephe", "path to the .se1 ephemeris files")
	out := flag.String("out", filepath.Join("testdata", "golden"), "output directory")
	flag.Parse()
	if *swetest == "" {
		log.Fatal("-swetest is required, see testdata/golden/README.md")
	}

	epheAbs, err := filepath.Abs(*ephe)
	if err != nil {
		log.Fatal(err)
	}

	for i, c := range charts {
		g, err := generate(*swetest, epheAbs, c.slug, c.datetimeUTC, c.lat, c.lon, c.houseSystem)
		if err != nil {
			log.Fatalf("%s: %v", c.slug, err)
		}
		name := fmt.Sprintf("%03d-%s.json", i+1, c.slug)
		buf, err := json.MarshalIndent(g, "", "  ")
		if err != nil {
			log.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(*out, name), append(buf, '\n'), 0o644); err != nil {
			log.Fatal(err)
		}
		fmt.Println(name)
	}
}

func generate(swetest, ephe, slug, datetimeUTC string, lat, lon float64, houseSystem string) (*goldenFile, error) {
	t, err := time.Parse(time.RFC3339, datetimeUTC)
	if err != nil {
		return nil, err
	}

	args := []string{
		"-edir" + ephe,
		fmt.Sprintf("-b%d.%d.%d", t.Day(), int(t.Month()), t.Year()),
		fmt.Sprintf("-ut%02d:%02d:%02d", t.Hour(), t.Minute(), t.Second()),
		"-p0123456789mAD",
		"-fPl", "-g,", "-head",
	}
	if houseSystem != chart.HouseSystemNone {
		hsys, ok := chart.HouseSystems[houseSystem]
		if !ok {
			return nil, fmt.Errorf("unknown house system %q", houseSystem)
		}
		args = append(args, fmt.Sprintf("-house%g,%g,%c", lon, lat, hsys))
	}

	raw, err := exec.Command(swetest, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("swetest: %v\n%s", err, raw)
	}

	g := &goldenFile{
		Name:   slug,
		Source: "swetest 2.10.03 (Astrodienst reference utility, same engine and data files as astro.com)",
	}
	g.Input = goldenInput{DatetimeUTC: datetimeUTC, Lat: lat, Lon: lon, HouseSystem: houseSystem}

	for _, line := range strings.Split(string(raw), "\n") {
		name, value, ok := strings.Cut(line, ",")
		if !ok {
			continue
		}
		// Unknown lines (e.g. the polar-fallback warning) are skipped.
		name = strings.TrimSpace(name)
		switch {
		case planetNames[name] != "":
			deg, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				return nil, fmt.Errorf("parse %q: %w", line, err)
			}
			g.Expected.Planets = append(g.Expected.Planets, goldenPlanet{Name: planetNames[name], Lon: deg})
		case strings.HasPrefix(name, "house "):
			num, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(name, "house ")))
			if err != nil {
				return nil, fmt.Errorf("parse %q: %w", line, err)
			}
			deg, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
			if err != nil {
				return nil, fmt.Errorf("parse %q: %w", line, err)
			}
			g.Expected.Houses = append(g.Expected.Houses, goldenHouse{Num: num, CuspLon: deg})
		}
	}

	if len(g.Expected.Planets) != len(planetNames) {
		return nil, fmt.Errorf("expected %d planets, parsed %d:\n%s", len(planetNames), len(g.Expected.Planets), raw)
	}
	if houseSystem != chart.HouseSystemNone && len(g.Expected.Houses) != 12 {
		return nil, fmt.Errorf("expected 12 houses, parsed %d:\n%s", len(g.Expected.Houses), raw)
	}
	return g, nil
}

// Package sweph is a thin cgo binding over the vendored Swiss Ephemeris
// C library (see README.md in this directory for provenance).
//
// The underlying library keeps global state and is not thread-safe, so every
// exported function serializes access through a package-level mutex.
package sweph

/*
#include <stdlib.h>
#include "swephexp.h"
*/
import "C"

import (
	"fmt"
	"sync"
	"time"
	"unsafe"
)

// Body identifies a celestial body using Swiss Ephemeris planet numbers.
type Body int

const (
	Sun        Body = C.SE_SUN
	Moon       Body = C.SE_MOON
	Mercury    Body = C.SE_MERCURY
	Venus      Body = C.SE_VENUS
	Mars       Body = C.SE_MARS
	Jupiter    Body = C.SE_JUPITER
	Saturn     Body = C.SE_SATURN
	Uranus     Body = C.SE_URANUS
	Neptune    Body = C.SE_NEPTUNE
	Pluto      Body = C.SE_PLUTO
	MeanNode   Body = C.SE_MEAN_NODE
	TrueNode   Body = C.SE_TRUE_NODE
	MeanApogee Body = C.SE_MEAN_APOG // "Lilith"
	Chiron     Body = C.SE_CHIRON
)

var mu sync.Mutex

// SetEphePath points the library at a directory containing .se1 ephemeris
// files. When set, calculations may use the Swiss ephemeris; otherwise the
// built-in Moshier approximation is used (no data files required).
func SetEphePath(path string) {
	mu.Lock()
	defer mu.Unlock()
	cs := C.CString(path)
	defer C.free(unsafe.Pointer(cs))
	C.swe_set_ephe_path(cs)
}

// Close releases resources held by the library (open ephemeris files).
func Close() {
	mu.Lock()
	defer mu.Unlock()
	C.swe_close()
}

// Version returns the Swiss Ephemeris library version, e.g. "2.10.03".
func Version() string {
	mu.Lock()
	defer mu.Unlock()
	var buf [C.AS_MAXCH]C.char
	C.swe_version(&buf[0])
	return C.GoString(&buf[0])
}

// JulDayUT converts a moment in time to a Julian day number in Universal Time.
func JulDayUT(t time.Time) float64 {
	t = t.UTC()
	hour := float64(t.Hour()) +
		float64(t.Minute())/60 +
		(float64(t.Second())+float64(t.Nanosecond())/1e9)/3600
	mu.Lock()
	defer mu.Unlock()
	return float64(C.swe_julday(
		C.int(t.Year()), C.int(int(t.Month())), C.int(t.Day()),
		C.double(hour), C.SE_GREG_CAL,
	))
}

// CalcResult holds the ecliptic position of a body.
type CalcResult struct {
	Lon      float64 // ecliptic longitude, degrees [0, 360)
	Lat      float64 // ecliptic latitude, degrees
	Dist     float64 // distance, AU
	LonSpeed float64 // longitude speed, degrees/day (negative = retrograde)
}

// CalcUT computes the position of a body at the given Julian day (UT).
// With useSwiss=true the Swiss ephemeris data files are used (SetEphePath
// must have been called); otherwise the Moshier approximation.
func CalcUT(jdUT float64, body Body, useSwiss bool) (CalcResult, error) {
	mu.Lock()
	defer mu.Unlock()
	var xx [6]C.double
	var serr [C.AS_MAXCH]C.char
	iflag := C.int32(C.SEFLG_SPEED)
	if useSwiss {
		iflag |= C.SEFLG_SWIEPH
	} else {
		iflag |= C.SEFLG_MOSEPH
	}
	ret := C.swe_calc_ut(C.double(jdUT), C.int32(body), iflag, &xx[0], &serr[0])
	if ret < 0 {
		return CalcResult{}, fmt.Errorf("swe_calc_ut(body=%d): %s", body, C.GoString(&serr[0]))
	}
	return CalcResult{
		Lon:      float64(xx[0]),
		Lat:      float64(xx[1]),
		Dist:     float64(xx[2]),
		LonSpeed: float64(xx[3]),
	}, nil
}

// HousesResult holds house cusps and the main chart angles.
type HousesResult struct {
	Cusps [13]float64 // indices 1..12 are used, index 0 is unused
	Asc   float64
	MC    float64
	// PolarFallback is true when the requested quadrant house system is
	// undefined at this latitude and the library fell back to Porphyry.
	PolarFallback bool
}

// HousesUT computes house cusps for the given Julian day (UT), geographic
// coordinates and house system code (e.g. 'P' Placidus, 'K' Koch,
// 'W' whole sign, 'E' equal, 'O' Porphyry).
func HousesUT(jdUT, lat, lon float64, hsys byte) HousesResult {
	mu.Lock()
	defer mu.Unlock()
	var cusps [13]C.double
	var ascmc [10]C.double
	ret := C.swe_houses(C.double(jdUT), C.double(lat), C.double(lon), C.int(hsys), &cusps[0], &ascmc[0])
	res := HousesResult{
		Asc:           float64(ascmc[C.SE_ASC]),
		MC:            float64(ascmc[C.SE_MC]),
		PolarFallback: ret < 0,
	}
	for i := range res.Cusps {
		res.Cusps[i] = float64(cusps[i])
	}
	return res
}

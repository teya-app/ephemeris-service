package chart

import "math"

var signNames = [12]string{
	"aries", "taurus", "gemini", "cancer", "leo", "virgo",
	"libra", "scorpio", "sagittarius", "capricorn", "aquarius", "pisces",
}

// normalizeLon maps any longitude to [0, 360).
func normalizeLon(lon float64) float64 {
	lon = math.Mod(lon, 360)
	if lon < 0 {
		lon += 360
	}
	return lon
}

// signFor returns the zodiac sign name for an ecliptic longitude.
func signFor(lon float64) string {
	return signNames[int(normalizeLon(lon)/30)%12]
}

// signLon returns the longitude within the sign, [0, 30).
func signLon(lon float64) float64 {
	return math.Mod(normalizeLon(lon), 30)
}

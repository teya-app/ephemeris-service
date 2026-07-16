package chart

// inArc reports whether longitude x lies in the arc going from `from`
// (inclusive) counter-clockwise to `to` (exclusive), handling the 0°/360°
// wrap.
func inArc(x, from, to float64) bool {
	x = normalizeLon(x)
	from = normalizeLon(from)
	to = normalizeLon(to)
	if from <= to {
		return x >= from && x < to
	}
	return x >= from || x < to
}

// houseFor returns the house number (1..12) containing the longitude,
// given cusps indexed 1..12. Returns 0 if cusps are degenerate.
func houseFor(lon float64, cusps [13]float64) int {
	for i := 1; i <= 12; i++ {
		next := i%12 + 1
		if inArc(lon, cusps[i], cusps[next]) {
			return i
		}
	}
	return 0
}

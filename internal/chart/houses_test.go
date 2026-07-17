package chart

import "testing"

func TestHouseFor(t *testing.T) {
	// Equal-house layout starting at 100°: cusp[i] = 100 + 30*(i-1),
	// so cusps 9..12 wrap past 360° (cusp 10 = 10°, cusp 11 = 40°, ...).
	var cusps [13]float64
	for i := 1; i <= 12; i++ {
		cusps[i] = normalizeLon(100 + 30*float64(i-1))
	}

	tests := []struct {
		lon  float64
		want int
	}{
		{100, 1},
		{129.99, 1},
		{130, 2},
		{5, 9},
		{15, 10},
		{99.99, 12},
	}
	for _, tt := range tests {
		if got := houseFor(tt.lon, cusps); got != tt.want {
			t.Errorf("houseFor(%v) = %d, want %d", tt.lon, got, tt.want)
		}
	}
}

// Non-uniform Placidus-style cusps with a 0° wrap between cusp 12 and cusp 1:
// house boundaries are inclusive at the cusp, exclusive at the next.
func TestHouseForRealisticCusps(t *testing.T) {
	var cusps [13]float64
	vals := []float64{0, 266.34, 299.02, 331.7, 4.38, 37.06, 69.74, 86.34, 119.02, 151.7, 184.38, 217.06, 249.74}
	copy(cusps[:], vals)

	tests := []struct {
		lon  float64
		want int
	}{
		{266.34, 1},  // exactly on the Ascendant cusp
		{280, 1},     // inside house 1
		{299.02, 2},  // exactly on cusp 2
		{350, 3},     // house 3 spans the 0° wrap (331.7 → 4.38)
		{0, 3},       // still house 3 just past 0°
		{20, 4},      // house 4
		{249.74, 12}, // exactly on cusp 12
		{249.73, 11}, // one hair before cusp 12
	}
	for _, tt := range tests {
		if got := houseFor(tt.lon, cusps); got != tt.want {
			t.Errorf("houseFor(%v) = %d, want %d", tt.lon, got, tt.want)
		}
	}
}

func TestInArc(t *testing.T) {
	tests := []struct {
		x, from, to float64
		want        bool
	}{
		{5, 0, 10, true},
		{0, 0, 10, true},
		{10, 0, 10, false},
		{355, 350, 10, true},
		{5, 350, 10, true},
		{20, 350, 10, false},
	}
	for _, tt := range tests {
		if got := inArc(tt.x, tt.from, tt.to); got != tt.want {
			t.Errorf("inArc(%v, %v, %v) = %v, want %v", tt.x, tt.from, tt.to, got, tt.want)
		}
	}
}

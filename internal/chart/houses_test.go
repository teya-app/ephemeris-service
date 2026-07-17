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

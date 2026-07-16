package chart

import "testing"

func TestAngularSep(t *testing.T) {
	tests := []struct {
		a, b, want float64
	}{
		{0, 0, 0},
		{0, 90, 90},
		{350, 10, 20}, // wrap through 0°
		{10, 350, 20}, // symmetric
		{0, 180, 180},
		{0, 181, 179}, // separation never exceeds 180
	}
	for _, tt := range tests {
		if got := angularSep(tt.a, tt.b); got != tt.want {
			t.Errorf("angularSep(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestComputeAspects(t *testing.T) {
	planets := []Planet{
		{Name: "sun", Lon: 10},
		{Name: "moon", Lon: 130.5},    // 120.5° from sun → trine, orb 0.5
		{Name: "mars", Lon: 15},       // 5° from sun → conjunction
		{Name: "mean_node", Lon: 190}, // excluded from aspects
		{Name: "venus", Lon: 250},     // 120° from moon → trine; 240° (=120) from sun → trine
	}
	aspects := computeAspects(planets)

	type key struct{ p1, p2, typ string }
	got := map[key]float64{}
	for _, a := range aspects {
		got[key{a.P1, a.P2, a.Type}] = a.Orb
	}

	want := []key{
		{"sun", "moon", "trine"},
		{"sun", "mars", "conjunction"},
		{"sun", "venus", "trine"},
		{"moon", "venus", "trine"},
	}
	for _, k := range want {
		if _, ok := got[k]; !ok {
			t.Errorf("missing aspect %v in %v", k, got)
		}
	}
	for k := range got {
		if k.p1 == "mean_node" || k.p2 == "mean_node" {
			t.Errorf("nodes must not participate in aspects: %v", k)
		}
	}
	if orb := got[key{"sun", "moon", "trine"}]; orb != 0.5 {
		t.Errorf("sun-moon trine orb = %v, want 0.5", orb)
	}
}

func TestComputeAspectsNoFalsePositives(t *testing.T) {
	// 45° separation is not a major aspect under our orbs.
	planets := []Planet{{Name: "sun", Lon: 0}, {Name: "moon", Lon: 45}}
	if aspects := computeAspects(planets); len(aspects) != 0 {
		t.Errorf("expected no aspects at 45°, got %v", aspects)
	}
}

package chart

import "testing"

func TestSignFor(t *testing.T) {
	tests := []struct {
		lon  float64
		want string
	}{
		{0, "aries"},
		{29.9999, "aries"},
		{30, "taurus"},
		{56.78, "taurus"},
		{180, "libra"},
		{359.9999, "pisces"},
		{360, "aries"},  // wraps
		{-10, "pisces"}, // negative wraps
		{725, "aries"},  // > 720 wraps twice
	}
	for _, tt := range tests {
		if got := signFor(tt.lon); got != tt.want {
			t.Errorf("signFor(%v) = %q, want %q", tt.lon, got, tt.want)
		}
	}
}

func TestSignLon(t *testing.T) {
	tests := []struct {
		lon  float64
		want float64
	}{
		{0, 0},
		{56.78, 26.78},
		{359.5, 29.5},
		{-0.5, 29.5},
	}
	for _, tt := range tests {
		got := signLon(tt.lon)
		if diff := got - tt.want; diff > 1e-9 || diff < -1e-9 {
			t.Errorf("signLon(%v) = %v, want %v", tt.lon, got, tt.want)
		}
	}
}

package server

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/teya-app/ephemeris-service/internal/chart"
)

func testHandler() http.Handler {
	log := slog.New(slog.DiscardHandler)
	return New(chart.NewEngine("", log), log)
}

func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/chart", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestChartOK(t *testing.T) {
	rec := post(t, testHandler(), `{
		"datetime_utc": "1990-05-17T21:15:00Z",
		"lat": 59.9386, "lon": 30.3141,
		"house_system": "placidus"
	}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var c chart.Chart
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if len(c.Planets) < 12 || len(c.Houses) != 12 || c.Angles == nil {
		t.Errorf("incomplete chart: %d planets, %d houses, angles=%v",
			len(c.Planets), len(c.Houses), c.Angles)
	}
	if c.Planets[0].Name != "sun" || c.Planets[0].Sign != "taurus" {
		t.Errorf("first planet = %+v, want sun in taurus", c.Planets[0])
	}
}

func TestChartDefaultsToPlacidus(t *testing.T) {
	rec := post(t, testHandler(), `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var c chart.Chart
	if err := json.Unmarshal(rec.Body.Bytes(), &c); err != nil {
		t.Fatal(err)
	}
	if c.Meta.HouseSystem != "placidus" {
		t.Errorf("house system = %q, want placidus", c.Meta.HouseSystem)
	}
}

func TestChartValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		// mustNotEcho is a submitted value that must not appear in the
		// error response (birth data never travels back).
		mustNotEcho string
	}{
		{"invalid json", `{`, ""},
		{"bad datetime", `{"datetime_utc": "17.05.1990", "lat": 0, "lon": 0}`, "17.05.1990"},
		{"year too small", `{"datetime_utc": "0900-01-01T00:00:00Z", "lat": 0, "lon": 0}`, "0900"},
		{"lat out of range", `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 91, "lon": 0}`, "1990"},
		{"lon out of range", `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 200}`, "1990"},
		{"bad house system", `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0, "house_system": "vedic"}`, "vedic"},
		{"unknown field", `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0, "name": "Ivan"}`, "Ivan"},
	}
	h := testHandler()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := post(t, h, tt.body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
			}
			if tt.mustNotEcho != "" && strings.Contains(rec.Body.String(), tt.mustNotEcho) {
				t.Errorf("error echoes user input %q: %s", tt.mustNotEcho, rec.Body.String())
			}
		})
	}
}

func TestHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	testHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["status"] != "ok" || resp["engine_version"] == "" {
		t.Errorf("unexpected healthz response: %v", resp)
	}
}

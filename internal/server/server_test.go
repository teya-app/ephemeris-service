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

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	log := slog.New(slog.DiscardHandler)
	engine, err := chart.NewEngine("", log)
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	return New(engine, log)
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
	rec := post(t, testHandler(t), `{
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
	rec := post(t, testHandler(t), `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0}`)
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
	h := testHandler(t)
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
	testHandler(t).ServeHTTP(rec, req)
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

func TestChartAcceptsOffsetDatetime(t *testing.T) {
	h := testHandler(t)
	utc := post(t, h, `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0}`)
	off := post(t, h, `{"datetime_utc": "1990-05-18T02:15:00+05:00", "lat": 0, "lon": 0}`)
	if utc.Code != http.StatusOK || off.Code != http.StatusOK {
		t.Fatalf("status = %d / %d, want 200", utc.Code, off.Code)
	}
	if utc.Body.String() != off.Body.String() {
		t.Error("same instant written with an offset must produce an identical chart")
	}
}

func TestChartRejectsTrailingData(t *testing.T) {
	rec := post(t, testHandler(t), `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0}{"x":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
}

func TestChartRejectsOversizedBody(t *testing.T) {
	body := `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0, "house_system": "` +
		strings.Repeat("x", 8<<10) + `"}`
	rec := post(t, testHandler(t), body)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413; body = %s", rec.Code, rec.Body.String())
	}
}

func TestMethodNotAllowedIsJSON(t *testing.T) {
	h := testHandler(t)
	for _, tt := range []struct{ method, path string }{
		{http.MethodGet, "/v1/chart"},
		{http.MethodPost, "/healthz"},
	} {
		req := httptest.NewRequest(tt.method, tt.path, nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("%s %s: status = %d, want 405", tt.method, tt.path, rec.Code)
		}
		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Errorf("%s %s: 405 body is not JSON: %s", tt.method, tt.path, rec.Body.String())
		} else if resp["error"] == "" {
			t.Errorf("%s %s: 405 body lacks error field: %v", tt.method, tt.path, resp)
		}
	}
}

func TestPolarFallbackAlwaysPresent(t *testing.T) {
	rec := post(t, testHandler(t), `{"datetime_utc": "1990-05-17T21:15:00Z", "lat": 0, "lon": 0}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	var meta map[string]any
	if err := json.Unmarshal(raw["meta"], &meta); err != nil {
		t.Fatal(err)
	}
	if _, ok := meta["polar_fallback"]; !ok {
		t.Error("meta.polar_fallback missing from response")
	}
}

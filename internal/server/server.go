// Package server exposes the chart engine over HTTP.
package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/teya-app/ephemeris-service/internal/chart"
)

const maxBodyBytes = 4 << 10 // requests are tiny; anything bigger is abuse

const (
	// Wide enough for natal charts and transits, fully inside the bundled
	// *_18.se1 file range (which starts mid-day 1800-01-01) and clear of
	// the pre-1582 calendar ambiguity.
	minYear = 1801
	maxYear = 2200
)

// Engine is the part of chart.Engine the server depends on.
type Engine interface {
	Compute(chart.Input) (*chart.Chart, error)
	Ephemeris() string
	EngineVersion() string
}

// Server routes HTTP requests to the chart engine.
type Server struct {
	engine Engine
	log    *slog.Logger
}

// New builds the HTTP handler with all routes and middleware attached.
func New(engine Engine, log *slog.Logger) http.Handler {
	s := &Server{engine: engine, log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /v1/chart", s.handleChart)
	// Method-less fallbacks keep 405 responses in the JSON error shape.
	mux.HandleFunc("/healthz", methodNotAllowed(http.MethodGet))
	mux.HandleFunc("/v1/chart", methodNotAllowed(http.MethodPost))
	return recoverMiddleware(log, requestLogMiddleware(log, mux))
}

type chartRequest struct {
	DatetimeUTC string  `json:"datetime_utc"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	HouseSystem string  `json:"house_system"`
}

// parse validates the raw request. Birth data is personal data: validation
// errors must not echo submitted values back (they end up in logs and
// client-side error trackers).
func (r chartRequest) parse() (chart.Input, error) {
	t, err := time.Parse(time.RFC3339, r.DatetimeUTC)
	if err != nil {
		return chart.Input{}, fmt.Errorf("datetime_utc: must be RFC3339, e.g. 2000-01-01T12:00:00Z")
	}
	t = t.UTC()
	if y := t.Year(); y < minYear || y > maxYear {
		return chart.Input{}, fmt.Errorf("datetime_utc: year must be within %d..%d", minYear, maxYear)
	}
	if r.Lat < -90 || r.Lat > 90 {
		return chart.Input{}, fmt.Errorf("lat: must be within -90..90")
	}
	if r.Lon < -180 || r.Lon > 180 {
		return chart.Input{}, fmt.Errorf("lon: must be within -180..180")
	}
	hsys := r.HouseSystem
	if hsys == "" {
		hsys = "placidus"
	}
	if _, ok := chart.HouseSystems[hsys]; !ok && hsys != chart.HouseSystemNone {
		return chart.Input{}, fmt.Errorf("house_system: unknown value")
	}
	return chart.Input{DatetimeUTC: t, Lat: r.Lat, Lon: r.Lon, HouseSystem: hsys}, nil
}

func (s *Server) handleChart(w http.ResponseWriter, r *http.Request) {
	var req chartRequest
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if dec.More() {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	in, err := req.parse()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	c, err := s.engine.Compute(in)
	if err != nil {
		// No birth data in logs — only the failure itself.
		s.log.Error("chart computation failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "computation failed")
		return
	}
	if err := writeJSON(w, http.StatusOK, c); err != nil {
		s.log.Error("response write failed", "error", err.Error())
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	if err := writeJSON(w, http.StatusOK, map[string]string{
		"status":         "ok",
		"engine_version": s.engine.EngineVersion(),
		"ephemeris":      s.engine.Ephemeris(),
	}); err != nil {
		s.log.Error("response write failed", "error", err.Error())
	}
}

// writeJSON marshals before writing the header: a marshal failure (e.g. a
// NaN in a chart) must yield a 500, not a 200 with a truncated body.
func writeJSON(w http.ResponseWriter, status int, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"internal error"}` + "\n"))
		return fmt.Errorf("marshal response: %w", err)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err = w.Write(append(b, '\n'))
	return err
}

func writeError(w http.ResponseWriter, status int, msg string) {
	_ = writeJSON(w, status, map[string]string{"error": msg})
}

func methodNotAllowed(allow string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

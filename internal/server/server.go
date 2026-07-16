// Package server exposes the chart engine over HTTP.
package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/teya-app/ephemeris-service/internal/chart"
)

const maxBodyBytes = 4 << 10 // requests are tiny; anything bigger is abuse

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
	if y := t.Year(); y < 1000 || y > 2999 {
		return chart.Input{}, fmt.Errorf("datetime_utc: year must be within 1000..2999")
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
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":         "ok",
		"engine_version": s.engine.EngineVersion(),
		"ephemeris":      s.engine.Ephemeris(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

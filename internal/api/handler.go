package api

import (
	"encoding/json"
	"net/http"
	"sync/atomic"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/fraud"
)

type Handler struct {
	engine atomic.Pointer[fraud.Engine]
}

func NewHandler(engine *fraud.Engine) *Handler {
	h := &Handler{}

	if engine != nil {
		h.engine.Store(engine)
	}

	return h
}

func (h *Handler) SetEngine(engine *fraud.Engine) {
	h.engine.Store(engine)
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ready", h.ready)
	mux.HandleFunc("POST /fraud-score", h.fraudScore)
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if h.engine.Load() == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready"))
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) fraudScore(w http.ResponseWriter, r *http.Request) {
	engine := h.engine.Load()
	if engine == nil {
		writeJSON(w, http.StatusOK, FraudScoreResponse{
			Approved:   true,
			FraudScore: 0.0,
		})
		return
	}

	var request FraudScoreRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, FraudScoreResponse{
			Approved:   true,
			FraudScore: 0.0,
		})
		return
	}

	result, err := engine.Score(request)
	if err != nil {
		writeJSON(w, http.StatusOK, FraudScoreResponse{
			Approved:   true,
			FraudScore: 0.0,
		})
		return
	}

	writeJSON(w, http.StatusOK, FraudScoreResponse{
		Approved:   result.Approved,
		FraudScore: result.FraudScore,
	})
}

func writeJSON(w http.ResponseWriter, status int, response FraudScoreResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

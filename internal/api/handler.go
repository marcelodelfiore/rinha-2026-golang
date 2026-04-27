package api

import (
	"encoding/json"
	"net/http"

	"github.com/marcelodelfiore/rinha-2026-golang/internal/detection"
)

type Handler struct {
	engine *detection.Engine
}

func NewHandler(engine *detection.Engine) *Handler {
	return &Handler{engine: engine}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /ready", h.ready)
	mux.HandleFunc("POST /fraud-score", h.fraudScore)
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (h *Handler) fraudScore(w http.ResponseWriter, r *http.Request) {
	var request FraudScoreRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, FraudScoreResponse{
			Approved:   true,
			FraudScore: 0.0,
		})
		return
	}

	result, err := h.engine.Score(request)
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

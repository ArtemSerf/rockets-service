package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"rockets/internal/libs/log"
	"rockets/internal/services/rockets"
	"rockets/internal/xerrors"
)

func NewHandler(service *rockets.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /messages", postMessage(service))
	mux.HandleFunc("GET /rockets/{channel}", getRocket(service))
	mux.HandleFunc("GET /rockets", listRockets(service))
	mux.HandleFunc("GET /health", health(service))
	return mux
}

func health(service *rockets.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := service.Ping(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeServiceError(ctx context.Context, w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, xerrors.ErrInvalidEvent), errors.Is(err, xerrors.ErrInvalidQuery):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, xerrors.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		log.Error(ctx, err, "request failed")
		writeError(w, http.StatusServiceUnavailable, "temporarily unavailable")
	}
}

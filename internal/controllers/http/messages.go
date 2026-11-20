package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"rockets/internal/services/rockets"
)

func postMessage(service *rockets.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 64<<10)

		var msg message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
			return
		}
		event, err := msg.toEvent()
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		duplicate, err := service.Ingest(ctx, event)
		switch {
		case err != nil:
			writeServiceError(ctx, w, err)
		case duplicate:
			writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		default:
			writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
		}
	}
}

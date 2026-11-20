package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"rockets/internal/models"
	"rockets/internal/services/rockets"
	"rockets/internal/xerrors"
)

func getRocket(service *rockets.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		rocket, err := service.Rocket(ctx, r.PathValue("channel"))
		if err != nil {
			writeServiceError(ctx, w, err)
			return
		}
		writeJSON(w, http.StatusOK, toRocketResponse(rocket))
	}
}

func listRockets(service *rockets.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		opts, err := parseListOptions(r.URL.Query())
		if err != nil {
			writeServiceError(ctx, w, err)
			return
		}
		list, err := service.Rockets(ctx, opts)
		if err != nil {
			writeServiceError(ctx, w, err)
			return
		}

		resp := make([]rocketJSON, 0, len(list))
		for _, rocket := range list {
			resp = append(resp, toRocketResponse(rocket))
		}
		writeJSON(w, http.StatusOK, map[string]any{"rockets": resp, "limit": opts.Limit, "offset": opts.Offset})
	}
}

func parseListOptions(q url.Values) (models.ListOptions, error) {
	opts := models.DefaultListOptions()
	for key, values := range q {
		if len(values) != 1 {
			return opts, fmt.Errorf("%w: %s given more than once", xerrors.ErrInvalidQuery, key)
		}
		var err error
		switch key {
		case "sort":
			opts.Sort = values[0]
		case "order":
			opts.Order = values[0]
		case "limit":
			opts.Limit, err = strconv.Atoi(values[0])
		case "offset":
			opts.Offset, err = strconv.Atoi(values[0])
		default:
			return opts, fmt.Errorf("%w: unknown parameter %q", xerrors.ErrInvalidQuery, key)
		}
		if err != nil {
			return opts, fmt.Errorf("%w: %s must be an integer", xerrors.ErrInvalidQuery, key)
		}
	}
	return opts, nil
}

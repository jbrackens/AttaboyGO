package handler

import (
	"encoding/json"
	"net/http"

	"github.com/attaboy/platform/internal/infra"
	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthHandler returns a health check endpoint.
func HealthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := infra.HealthCheck(r.Context(), pool)
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unhealthy",
				"error":  err.Error(),
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
		})
	}
}

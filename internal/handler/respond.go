package handler

import (
	"encoding/json"
	"net/http"

	"github.com/attaboy/platform/internal/domain"
)

// RespondJSON writes a JSON response with the given status code.
func RespondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

// RespondError writes a JSON error response, detecting domain.AppError for status codes.
func RespondError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*domain.AppError); ok {
		RespondJSON(w, appErr.Status, map[string]string{
			"code":    appErr.Code,
			"message": appErr.Message,
		})
		return
	}
	RespondJSON(w, http.StatusInternalServerError, map[string]string{
		"code":    "INTERNAL_ERROR",
		"message": "internal server error",
	})
}

// DecodeJSON reads and decodes a JSON request body into dst.
func DecodeJSON(r *http.Request, dst interface{}) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

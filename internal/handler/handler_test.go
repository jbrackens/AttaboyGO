package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/attaboy/platform/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- RespondJSON Tests ---

func TestRespondJSON(t *testing.T) {
	t.Run("200 with body", func(t *testing.T) {
		w := httptest.NewRecorder()
		RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		assert.Equal(t, http.StatusOK, w.Code)
		var body map[string]string
		require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
		assert.Equal(t, "ok", body["status"])
	})

	t.Run("201 with body", func(t *testing.T) {
		w := httptest.NewRecorder()
		RespondJSON(w, http.StatusCreated, map[string]int{"id": 42})
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("204 with nil body", func(t *testing.T) {
		w := httptest.NewRecorder()
		RespondJSON(w, http.StatusNoContent, nil)
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Empty(t, w.Body.String())
	})
}

// --- RespondError Tests ---

func TestRespondError(t *testing.T) {
	t.Run("AppError maps to correct status", func(t *testing.T) {
		tests := []struct {
			err        *domain.AppError
			wantStatus int
			wantCode   string
		}{
			{domain.ErrNotFound("player", "123"), 404, "NOT_FOUND"},
			{domain.ErrValidation("bad input"), 400, "VALIDATION_ERROR"},
			{domain.ErrUnauthorized("no token"), 401, "UNAUTHORIZED"},
			{domain.ErrForbidden("not allowed"), 403, "FORBIDDEN"},
			{domain.ErrConflict("duplicate"), 409, "CONFLICT"},
			{domain.ErrInsufficientBalance(), 400, "INSUFFICIENT_BALANCE"},
			{domain.ErrAccountLocked("locked"), 429, "ACCOUNT_LOCKED"},
			{domain.ErrInternal("oops", nil), 500, "INTERNAL_ERROR"},
		}

		for _, tt := range tests {
			t.Run(tt.wantCode, func(t *testing.T) {
				w := httptest.NewRecorder()
				RespondError(w, tt.err)
				assert.Equal(t, tt.wantStatus, w.Code)

				var body map[string]string
				require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
				assert.Equal(t, tt.wantCode, body["code"])
			})
		}
	})

	t.Run("generic error returns 500", func(t *testing.T) {
		w := httptest.NewRecorder()
		RespondError(w, assert.AnError)
		assert.Equal(t, http.StatusInternalServerError, w.Code)

		var body map[string]string
		require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
		assert.Equal(t, "INTERNAL_ERROR", body["code"])
		assert.Equal(t, "internal server error", body["message"])
	})
}

// --- DecodeJSON Tests ---

func TestDecodeJSON(t *testing.T) {
	t.Run("valid JSON body", func(t *testing.T) {
		body := bytes.NewBufferString(`{"name":"test","value":42}`)
		r := httptest.NewRequest(http.MethodPost, "/", body)
		var dst struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}
		require.NoError(t, DecodeJSON(r, &dst))
		assert.Equal(t, "test", dst.Name)
		assert.Equal(t, 42, dst.Value)
	})

	t.Run("invalid JSON returns error", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid`)
		r := httptest.NewRequest(http.MethodPost, "/", body)
		var dst map[string]interface{}
		err := DecodeJSON(r, &dst)
		require.Error(t, err)
	})

	t.Run("body exceeding 1MiB returns error", func(t *testing.T) {
		// Create a body > 1 MiB
		bigBody := strings.Repeat("x", 1<<20+1)
		r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(bigBody))
		var dst map[string]interface{}
		err := DecodeJSON(r, &dst)
		require.Error(t, err)
	})
}

// --- ClientIP Tests ---

func TestClientIP(t *testing.T) {
	t.Run("X-Forwarded-For single IP", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Forwarded-For", "1.2.3.4")
		assert.Equal(t, "1.2.3.4", ClientIP(r))
	})

	t.Run("X-Forwarded-For multiple IPs takes first", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8, 9.10.11.12")
		assert.Equal(t, "1.2.3.4", ClientIP(r))
	})

	t.Run("X-Forwarded-For with spaces", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Forwarded-For", "  1.2.3.4  ")
		assert.Equal(t, "1.2.3.4", ClientIP(r))
	})

	t.Run("no X-Forwarded-For uses RemoteAddr", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "10.0.0.1:54321"
		assert.Equal(t, "10.0.0.1", ClientIP(r))
	})

	t.Run("RemoteAddr without port", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.RemoteAddr = "10.0.0.1"
		// No colon, returns full string
		assert.Equal(t, "10.0.0.1", ClientIP(r))
	})
}

// --- RequestID Middleware Tests ---

func TestRequestID(t *testing.T) {
	t.Run("generates ID when none provided", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := GetRequestID(r.Context())
			assert.NotEmpty(t, id)
			w.WriteHeader(http.StatusOK)
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
	})

	t.Run("uses provided X-Request-ID", func(t *testing.T) {
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := GetRequestID(r.Context())
			assert.Equal(t, "my-custom-id", id)
			w.WriteHeader(http.StatusOK)
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.Header.Set("X-Request-ID", "my-custom-id")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Equal(t, "my-custom-id", w.Header().Get("X-Request-ID"))
	})
}

func TestGetRequestID_EmptyContext(t *testing.T) {
	id := GetRequestID(context.Background())
	assert.Empty(t, id)
}

// --- JSONContentType Middleware Tests ---

func TestJSONContentType(t *testing.T) {
	handler := JSONContentType(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

// --- CORS Middleware Tests ---

func TestCORSWithOrigins(t *testing.T) {
	t.Run("sets CORS headers", func(t *testing.T) {
		handler := CORSWithOrigins("*")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
		assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
		assert.Contains(t, w.Header().Get("Access-Control-Expose-Headers"), "X-Request-ID")
	})

	t.Run("OPTIONS returns 204", func(t *testing.T) {
		handler := CORSWithOrigins("https://example.com")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		r := httptest.NewRequest(http.MethodOptions, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))
	})

	t.Run("specific origin", func(t *testing.T) {
		handler := CORSWithOrigins("https://app.attaboy.io")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Equal(t, "https://app.attaboy.io", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

// --- Recovery Middleware Tests ---

func TestRecovery(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		logger := noopLogger()
		handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something went wrong")
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		// Should not panic
		assert.NotPanics(t, func() {
			handler.ServeHTTP(w, r)
		})

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "INTERNAL_ERROR")
	})

	t.Run("passes through without panic", func(t *testing.T) {
		logger := noopLogger()
		handler := Recovery(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"ok":true}`))
		}))

		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// --- responseWriter Tests ---

func TestResponseWriter_CapturesStatus(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, status: 200}

	rw.WriteHeader(http.StatusNotFound)
	assert.Equal(t, 404, rw.status)
	assert.Equal(t, 404, w.Code)
}

// helper

func noopLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

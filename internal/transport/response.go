package transport

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/tx-mi/bazgain-places-go/internal/places"
	"github.com/tx-mi/bazgain-places-go/internal/wire"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
func writeError(logger *slog.Logger, w http.ResponseWriter, r *http.Request, err error) {
	var appErr *places.AppError
	if !errors.As(err, &appErr) {
		appErr = places.Internal(err)
	}
	fields := make([]wire.FieldError, 0, len(appErr.Fields))
	for _, field := range appErr.Fields {
		fields = append(fields, wire.FieldError{Field: field.Field, Code: field.Code, Message: field.Message})
	}
	if appErr.Status >= 500 {
		logger.ErrorContext(r.Context(), "request failed", "request_id", requestID(r.Context()), "error_code", appErr.Code, "error", appErr.Cause)
	}
	writeJSON(w, appErr.Status, wire.ErrorEnvelope{Error: wire.ErrorBody{Code: appErr.Code, Message: appErr.Message, Fields: fields}})
}

func writePublic(w http.ResponseWriter, r *http.Request, etag string, maxAgeSeconds int, payload any) {
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAgeSeconds))
	w.Header().Set("Vary", "Accept-Encoding, Origin")
	if matchesETag(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func matchesETag(header, current string) bool {
	header = strings.TrimSpace(header)
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		if strings.HasPrefix(candidate, "W/") {
			candidate = strings.TrimSpace(strings.TrimPrefix(candidate, "W/"))
		}
		if unquoteETag(candidate) == unquoteETag(current) {
			return true
		}
	}
	return false
}

func unquoteETag(value string) string {
	if len(value) >= 2 && strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		return value[1 : len(value)-1]
	}
	return value
}

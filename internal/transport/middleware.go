package transport

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tx-mi/bazgain-places-go/internal/places"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func requestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

func middleware(logger *slog.Logger, maxBody int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		id := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if id == "" || len(id) > 128 {
			id = uuid.NewString()
		}
		w.Header().Set("X-Request-ID", id)
		r = r.WithContext(context.WithValue(r.Context(), requestIDKey, id))
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBody)
		}
		recorder := &statusRecorder{ResponseWriter: w}
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.ErrorContext(r.Context(), "panic recovered", "request_id", id, "panic", recovered, "stack", string(debug.Stack()))
				if recorder.status == 0 {
					writeError(logger, recorder, r, places.Internal(nil))
				}
			}
			status := recorder.status
			if status == 0 {
				status = http.StatusOK
			}
			logger.InfoContext(r.Context(), "http request", "request_id", id, "method", r.Method, "path", r.URL.Path, "status", status, "duration_ms", time.Since(started).Milliseconds(), "bytes", recorder.bytes)
		}()
		next.ServeHTTP(recorder, r)
	})
}

func clientIP(r *http.Request, trusted []*net.IPNet) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if !trustedIP(peer, trusted) {
		return host
	}

	forwardedHeader := r.Header.Get("X-Forwarded-For")
	// Avoid allocating an unbounded slice even when the direct peer is trusted.
	// The peer address is a safe fallback when a proxy sends a malformed header.
	if len(forwardedHeader) > 4096 {
		return host
	}
	forwarded := strings.Split(forwardedHeader, ",")
	var leftmost net.IP
	for i := len(forwarded) - 1; i >= 0; i-- {
		candidate := net.ParseIP(strings.TrimSpace(forwarded[i]))
		if candidate == nil {
			continue
		}
		leftmost = candidate
		if !trustedIP(candidate, trusted) {
			return candidate.String()
		}
	}
	if leftmost != nil {
		return leftmost.String()
	}
	return host
}

func trustedIP(ip net.IP, trusted []*net.IPNet) bool {
	if ip == nil {
		return false
	}
	for _, network := range trusted {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

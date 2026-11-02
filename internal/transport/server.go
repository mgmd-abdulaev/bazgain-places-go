package transport

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/tx-mi/bazgain-places-go/internal/places"
)

type Authenticator interface {
	Authenticate(context.Context, string) (places.Actor, error)
}
type Limiter interface {
	Allow(context.Context, string, string, int64, time.Duration) (bool, time.Duration, error)
}
type Health interface {
	Postgres(context.Context) error
	Redis(context.Context) error
}

type Server struct {
	service                 *places.Service
	auth                    Authenticator
	limiter                 Limiter
	health                  Health
	logger                  *slog.Logger
	trusted                 []*net.IPNet
	publicLimit, adminLimit int64
	window, cacheTTL        time.Duration
	maxBody                 int64
}

type Options struct {
	TrustedProxies          []*net.IPNet
	PublicLimit, AdminLimit int64
	RateWindow, CacheTTL    time.Duration
	MaxBodyBytes            int64
}

func NewServer(service *places.Service, auth Authenticator, limiter Limiter, health Health, logger *slog.Logger, options Options) http.Handler {
	s := &Server{service: service, auth: auth, limiter: limiter, health: health, logger: logger, trusted: options.TrustedProxies, publicLimit: options.PublicLimit, adminLimit: options.AdminLimit, window: options.RateWindow, cacheTTL: options.CacheTTL, maxBody: options.MaxBodyBytes}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/place-collections", s.requireMethod(http.MethodGet, s.public("collections", s.collections)))
	mux.HandleFunc("/v1/places", s.requireMethod(http.MethodGet, s.public("places-list", s.list)))
	mux.HandleFunc("/v1/places/{placeSlug}", s.requireMethod(http.MethodGet, s.public("place-detail", s.detail)))
	mux.HandleFunc("/v1/admin/places/{id}/publish", s.requireMethod(http.MethodPost, s.admin("place-publish", s.publish)))
	mux.HandleFunc("/v1/admin/places/{id}/archive", s.requireMethod(http.MethodPost, s.admin("place-archive", s.archive)))
	mux.HandleFunc("/v1/health/live", s.requireMethod(http.MethodGet, s.live))
	mux.HandleFunc("/v1/health/ready", s.requireMethod(http.MethodGet, s.ready))
	mux.HandleFunc("/", s.notFound)
	return middleware(logger, options.MaxBodyBytes, mux)
}

func (s *Server) requireMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			w.Header().Set("Allow", method)
			writeError(s.logger, w, r, places.MethodNotAllowed())
			return
		}
		next(w, r)
	}
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	writeError(s.logger, w, r, places.RouteNotFound())
}

func (s *Server) public(scope string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, retry, err := s.limiter.Allow(r.Context(), scope, clientIP(r, s.trusted), s.publicLimit, s.window)
		if err != nil {
			writeError(s.logger, w, r, places.Internal(err))
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", retryAfterSeconds(retry))
			writeError(s.logger, w, r, places.RateLimited())
			return
		}
		next(w, r)
	}
}
func (s *Server) admin(scope string, next func(http.ResponseWriter, *http.Request, places.Actor)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, retry, err := s.limiter.Allow(r.Context(), scope, clientIP(r, s.trusted), s.adminLimit, s.window)
		if err != nil {
			writeError(s.logger, w, r, places.Internal(err))
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", retryAfterSeconds(retry))
			writeError(s.logger, w, r, places.RateLimited())
			return
		}
		actor, err := s.auth.Authenticate(r.Context(), r.Header.Get("Authorization"))
		if err != nil {
			writeError(s.logger, w, r, err)
			return
		}
		if r.Body != nil {
			if _, err := io.Copy(io.Discard, r.Body); err != nil {
				var maxErr *http.MaxBytesError
				if errors.As(err, &maxErr) {
					writeError(s.logger, w, r, &places.AppError{Status: http.StatusRequestEntityTooLarge, Code: "payload_too_large", Message: "Payload too large"})
					return
				}
				writeError(s.logger, w, r, places.Internal(err))
				return
			}
		}
		next(w, r, actor)
	}
}

func retryAfterSeconds(retry time.Duration) string {
	seconds := (retry + time.Second - 1) / time.Second
	if seconds < 1 {
		seconds = 1
	}
	return strconv.FormatInt(int64(seconds), 10)
}

func (s *Server) collections(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.Collections(r.Context())
	if err != nil {
		writeError(s.logger, w, r, err)
		return
	}
	writePublic(w, r, result.ETag, int(s.cacheTTL.Seconds()), result.Payload)
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.List(r.Context())
	if err != nil {
		writeError(s.logger, w, r, err)
		return
	}
	writePublic(w, r, result.ETag, int(s.cacheTTL.Seconds()), result.Payload)
}
func (s *Server) detail(w http.ResponseWriter, r *http.Request) {
	result, err := s.service.Detail(r.Context(), r.PathValue("placeSlug"))
	if err != nil {
		writeError(s.logger, w, r, err)
		return
	}
	writePublic(w, r, result.ETag, int(s.cacheTTL.Seconds()), result.Payload)
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func validID(raw string) (string, error) {
	if !uuidPattern.MatchString(raw) {
		return "", places.Validation("id", "invalid_uuid", "Must be a UUID")
	}
	return strings.ToLower(raw), nil
}
func (s *Server) publish(w http.ResponseWriter, r *http.Request, actor places.Actor) {
	id, err := validID(r.PathValue("id"))
	if err != nil {
		writeError(s.logger, w, r, err)
		return
	}
	env := s.auditEnvelope(r)
	result, err := s.service.Publish(r.Context(), id, actor, env)
	if err != nil {
		writeError(s.logger, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) archive(w http.ResponseWriter, r *http.Request, actor places.Actor) {
	id, err := validID(r.PathValue("id"))
	if err != nil {
		writeError(s.logger, w, r, err)
		return
	}
	env := s.auditEnvelope(r)
	result, err := s.service.Archive(r.Context(), id, actor, env)
	if err != nil {
		writeError(s.logger, w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
func (s *Server) auditEnvelope(r *http.Request) places.AuditEnvelope {
	ip := clientIP(r, s.trusted)
	var ipValue *string
	if ip != "" {
		ipValue = &ip
	}
	ua := sanitizeAuditHeader(r.UserAgent(), 512)
	var userAgent *string
	if ua != "" {
		userAgent = &ua
	}
	return places.AuditEnvelope{IP: ipValue, UserAgent: userAgent}
}

// sanitizeAuditHeader removes control characters before storing a request header in
// the audit log and truncates without splitting a UTF-8 encoded rune.
func sanitizeAuditHeader(value string, maxBytes int) string {
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, strings.ToValidUTF8(value, ""))
	if len(value) <= maxBytes {
		return value
	}
	end := maxBytes
	for end > 0 && !utf8.ValidString(value[:end]) {
		end--
	}
	return value[:end]
}
func (s *Server) live(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.health.Postgres(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error"})
		return
	}
	if err := s.health.Redis(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Package pilot serves the authenticated local-pilot API.
package pilot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct{ Origin, JWTKey, PreviewKey, LineChannelID string }
type Server struct {
	db     *pgxpool.Pool
	config Config
	now    func() time.Time
	verify func(context.Context, string) (string, error)
}
type apiError struct {
	status  int
	code    string
	details []any
}

func (e *apiError) Error() string { return e.code }
func bad(field string) error {
	return &apiError{422, "VALIDATION_ERROR", []any{gin.H{"field": field, "code": "INVALID", "message": "ข้อมูลไม่ถูกต้อง"}}}
}
func conflict(code string, details ...any) error { return &apiError{409, code, details} }
func missing() error                             { return &apiError{404, "NOT_FOUND", nil} }
func unauth() error                              { return &apiError{401, "UNAUTHENTICATED", nil} }

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func New(db *pgxpool.Pool, config Config) (*Server, error) {
	u, err := url.Parse(config.Origin)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"))) {
		return nil, errors.New("WEB_APP_URL must be an exact HTTPS origin (HTTP allowed only on loopback)")
	}
	if len(config.JWTKey) < 32 || len(config.PreviewKey) < 32 || config.JWTKey == config.PreviewKey || config.LineChannelID == "" {
		return nil, errors.New("independent signing keys of at least 32 bytes and LINE_LOGIN_CHANNEL_ID are required")
	}
	s := &Server{db: db, config: config, now: time.Now}
	s.verify = s.verifyLine
	return s, nil
}

func fail(c *gin.Context, status int, code string, details ...any) {
	if details == nil {
		details = []any{}
	}
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": http.StatusText(status), "details": details}})
}
func (s *Server) endpoint(fn func(*gin.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := fn(c); err != nil {
			var a *apiError
			if errors.As(err, &a) {
				fail(c, a.status, a.code, a.details...)
			} else {
				slog.Error("pilot request failed", "route", c.FullPath(), "request_id", c.GetString("request_id"))
				fail(c, 503, "SERVICE_UNAVAILABLE")
			}
		}
	}
}
func (s *Server) Register(router *gin.Engine) {
	api := router.Group("/api/v1")
	api.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		requestID := randomUUID()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})
	api.POST("/auth/line", s.origin, s.endpoint(s.login))
	api.POST("/auth/refresh", s.origin, s.endpoint(s.refresh))
	api.POST("/auth/logout", s.origin, s.endpoint(s.logout))
	protected := api.Group("")
	protected.Use(s.requireAuth)
	protected.GET("/me", s.endpoint(s.getProfile))
	protected.GET("/me/profile", s.endpoint(s.getProfile))
	protected.PUT("/me/profile", s.endpoint(s.putProfile))
	protected.GET("/platforms", s.endpoint(s.platforms))
	protected.GET("/vehicles", s.endpoint(s.vehicles))
	protected.POST("/vehicles", s.endpoint(s.createVehicle))
	protected.PATCH("/vehicles/:vehicle_id", s.endpoint(s.patchVehicle))
	protected.POST("/sessions/preview", s.endpoint(s.preview))
	protected.POST("/sessions", s.endpoint(s.confirm))
	protected.GET("/sessions", s.endpoint(s.history))
	protected.POST("/sessions/:session_id/preview", s.endpoint(s.preview))
	protected.PATCH("/sessions/:session_id", s.endpoint(s.confirm))
	protected.GET("/sessions/:session_id", s.endpoint(s.getSession))
	protected.DELETE("/sessions/:session_id", s.endpoint(s.deleteSession))
	protected.GET("/reports/daily", s.endpoint(s.dailyReport))

}
func (s *Server) origin(c *gin.Context) {
	if c.GetHeader("Origin") != s.config.Origin {
		fail(c, 403, "ORIGIN_FORBIDDEN")
		return
	}
	c.Next()
}
func ok(c *gin.Context, status int, value any) { c.JSON(status, gin.H{"data": value}) }
func (s *Server) transaction(c *gin.Context, fn func(pgx.Tx) error) error {
	tx, e := s.db.Begin(c.Request.Context())
	if e != nil {
		return e
	}
	defer tx.Rollback(c.Request.Context())
	if e = fn(tx); e != nil {
		return e
	}
	return tx.Commit(c.Request.Context())
}
func uid(c *gin.Context) string { return c.GetString("user_id") }

var uuidRE = regexp.MustCompile(`^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12}$`)

func id(c *gin.Context, name string) (string, error) {
	v := c.Param(name)
	if !uuidRE.MatchString(v) {
		return "", bad(name)
	}
	return strings.ToLower(v), nil
}
func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	v := hex.EncodeToString(b[:])
	return v[:8] + "-" + v[8:12] + "-" + v[12:16] + "-" + v[16:20] + "-" + v[20:]
}

func body(c *gin.Context, allowed, required string) (map[string]any, error) {
	data, e := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 256<<10))
	if e != nil {
		return nil, bad("body")
	}
	if !utf8.Valid(data) {
		return nil, bad("body")
	}
	var m map[string]any
	if e = json.Unmarshal(data, &m); e != nil || m == nil {
		return nil, &apiError{400, "BAD_REQUEST", nil}
	}
	if e = keys(m, allowed, required); e != nil {
		return nil, e
	}
	return m, nil
}
func keys(m map[string]any, allowed, required string) error {
	allowedKeys := map[string]bool{}
	for _, name := range strings.Fields(allowed) {
		allowedKeys[name] = true
	}
	for k, v := range m {
		if !allowedKeys[k] || v == nil {
			return bad(k)
		}
	}
	for _, k := range strings.Fields(required) {
		if _, found := m[k]; !found {
			return bad(k)
		}
	}
	return nil
}
func str(m map[string]any, k string) (string, error) {
	v, ok := m[k].(string)
	if !ok {
		return "", bad(k)
	}
	return v, nil
}
func length(v string, max int) bool { return utf8.RuneCountInString(v) <= max }
func pgCode(e error, code string) bool {
	var p *pgconn.PgError
	return errors.As(e, &p) && p.Code == code
}

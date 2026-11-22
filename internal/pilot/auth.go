package pilot

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// LINE verifies the signature; this service additionally checks the returned claims.
// The endpoint is fixed, and production has no configurable verifier/bypass switch.
func (s *Server) verifyLine(ctx context.Context, token string) (string, error) {
	values := url.Values{"id_token": {token}, "client_id": {s.config.LineChannelID}}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.line.me/oauth2/v2.1/verify", strings.NewReader(values.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		return "", &apiError{503, "SERVICE_UNAVAILABLE", nil}
	}
	defer response.Body.Close()
	if response.StatusCode >= 500 || response.StatusCode == 429 {
		return "", &apiError{503, "SERVICE_UNAVAILABLE", nil}
	}
	if response.StatusCode != 200 {
		return "", unauth()
	}
	var claims struct {
		Issuer   string `json:"iss"`
		Audience string `json:"aud"`
		Subject  string `json:"sub"`
		Expires  int64  `json:"exp"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 65536)).Decode(&claims); err != nil {
		return "", unauth()
	}
	if claims.Issuer != "https://access.line.me" || claims.Audience != s.config.LineChannelID || claims.Subject == "" || len(claims.Subject) > 128 || claims.Expires <= s.now().Unix() {
		return "", unauth()
	}
	return claims.Subject, nil
}
func sign(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func signed(value any, key string) string {
	raw, _ := json.Marshal(value)
	payload := base64.RawURLEncoding.EncodeToString(raw)
	return payload + "." + sign([]byte(payload), key)
}
func verifySigned(token, key string, out any) error {
	parts := strings.Split(token, ".")
	if len(parts) != 2 || len(token) > 16384 || !hmac.Equal([]byte(sign([]byte(parts[0]), key)), []byte(parts[1])) {
		return errors.New("invalid signature")
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[0])
	if e != nil {
		return e
	}
	return json.Unmarshal(raw, out)
}

type accessClaims struct {
	User    string `json:"sub"`
	Session string `json:"sid"`
	Expires int64  `json:"exp"`
}

const jwtHeader = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

func (s *Server) access(user, session string, expiry time.Time) (string, int) {
	exp := s.now().Add(15 * time.Minute)
	if expiry.Before(exp) {
		exp = expiry
	}
	raw, _ := json.Marshal(accessClaims{user, session, exp.Unix()})
	p := jwtHeader + "." + base64.RawURLEncoding.EncodeToString(raw)
	return p + "." + sign([]byte(p), s.config.JWTKey), int(exp.Unix() - s.now().Unix())
}
func (s *Server) readAccess(token string) (accessClaims, error) {
	var claims accessClaims
	parts := strings.Split(token, ".")
	if len(parts) != 3 || parts[0] != jwtHeader || len(token) > 4096 || !hmac.Equal([]byte(sign([]byte(parts[0]+"."+parts[1]), s.config.JWTKey)), []byte(parts[2])) {
		return claims, unauth()
	}
	raw, e := base64.RawURLEncoding.DecodeString(parts[1])
	if e != nil {
		return claims, unauth()
	}
	if json.Unmarshal(raw, &claims) != nil || claims.Expires <= s.now().Unix() || !uuidRE.MatchString(claims.User) || !uuidRE.MatchString(claims.Session) {
		return claims, unauth()
	}
	return claims, nil
}
func (s *Server) requireAuth(c *gin.Context) {
	h := strings.Split(c.GetHeader("Authorization"), " ")
	if len(h) != 2 || h[0] != "Bearer" {
		fail(c, 401, "UNAUTHENTICATED")
		return
	}
	claims, e := s.readAccess(h[1])
	if e != nil {
		fail(c, 401, "UNAUTHENTICATED")
		return
	}
	var exists bool
	e = s.db.QueryRow(c.Request.Context(), `SELECT EXISTS(SELECT FROM auth_sessions a JOIN users u ON u.id=a.user_id WHERE a.id=$1 AND a.user_id=$2 AND a.expires_at>$3 AND a.revoked_at IS NULL AND u.deleted_at IS NULL)`, claims.Session, claims.User, s.now()).Scan(&exists)
	if e != nil {
		fail(c, 503, "SERVICE_UNAVAILABLE")
		return
	}
	if !exists {
		fail(c, 401, "UNAUTHENTICATED")
		return
	}
	c.Set("user_id", claims.User)
	c.Set("session_id", claims.Session)
	c.Next()
}
func hashCookie(c *gin.Context) []byte {
	value, e := c.Cookie("__Host-rydo_refresh")
	if e != nil || len(value) > 512 {
		return nil
	}
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}
func (s *Server) setCookie(c *gin.Context, value string, expiry time.Time) {
	age := int(expiry.Unix() - s.now().Unix())
	if value == "" {
		age = -1
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: "__Host-rydo_refresh", Value: value, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: age, Expires: expiry})
}
func (s *Server) authResponse(c *gin.Context, user, session string, expiry time.Time) error {
	p, e := s.profile(c.Request.Context(), s.db, user)
	if e != nil {
		return e
	}
	token, seconds := s.access(user, session, expiry)
	ok(c, 200, gin.H{"access_token": token, "token_type": "Bearer", "expires_in": seconds, "user": gin.H{"id": p.ID, "display_name": p.DisplayName}, "onboarding_completed": p.Onboarded})
	return nil
}
func (s *Server) login(c *gin.Context) error {
	m, e := body(c, "id_token", "id_token")
	if e != nil {
		return e
	}
	token, e := str(m, "id_token")
	if e != nil || token == "" || len(token) > 16384 {
		return bad("id_token")
	}
	subject, e := s.verify(c.Request.Context(), token)
	if e != nil {
		var a *apiError
		if errors.As(e, &a) {
			return e
		}
		return unauth()
	}
	var credential [32]byte
	if _, e = rand.Read(credential[:]); e != nil {
		return e
	}
	refresh := base64.RawURLEncoding.EncodeToString(credential[:])
	hash := sha256.Sum256([]byte(refresh))
	expiry := s.now().Add(30 * 24 * time.Hour)
	var user, session string
	e = s.transaction(c, func(tx pgx.Tx) error {
		if err := tx.QueryRow(c.Request.Context(), `INSERT INTO users(line_user_id) VALUES($1) ON CONFLICT(line_user_id) DO UPDATE SET updated_at=now() WHERE users.deleted_at IS NULL RETURNING id::text`, subject).Scan(&user); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return unauth()
			}
			return err
		}
		if old := hashCookie(c); old != nil {
			if _, err := tx.Exec(c.Request.Context(), `UPDATE auth_sessions SET revoked_at=$2 WHERE refresh_token_hash=$1 AND revoked_at IS NULL`, old, s.now()); err != nil {
				return err
			}
		}
		return tx.QueryRow(c.Request.Context(), `INSERT INTO auth_sessions(user_id,refresh_token_hash,expires_at) VALUES($1,$2,$3) RETURNING id::text`, user, hash[:], expiry).Scan(&session)
	})
	if e != nil {
		return e
	}
	s.setCookie(c, refresh, expiry)
	return s.authResponse(c, user, session, expiry)
}
func (s *Server) refresh(c *gin.Context) error {
	hash := hashCookie(c)
	if hash == nil {
		return unauth()
	}
	var user, session string
	var expiry time.Time
	e := s.db.QueryRow(c.Request.Context(), `SELECT a.user_id::text,a.id::text,a.expires_at FROM auth_sessions a JOIN users u ON u.id=a.user_id WHERE a.refresh_token_hash=$1 AND a.revoked_at IS NULL AND a.expires_at>$2 AND u.deleted_at IS NULL`, hash, s.now()).Scan(&user, &session, &expiry)
	if errors.Is(e, pgx.ErrNoRows) {
		return unauth()
	}
	if e != nil {
		return e
	}
	value, _ := c.Cookie("__Host-rydo_refresh")
	s.setCookie(c, value, expiry)
	return s.authResponse(c, user, session, expiry)
}
func (s *Server) logout(c *gin.Context) error {
	if hash := hashCookie(c); hash != nil {
		if _, e := s.db.Exec(c.Request.Context(), `UPDATE auth_sessions SET revoked_at=$2 WHERE refresh_token_hash=$1 AND revoked_at IS NULL`, hash, s.now()); e != nil {
			return e
		}
	}
	s.setCookie(c, "", time.Unix(1, 0))
	c.Status(204)
	return nil
}

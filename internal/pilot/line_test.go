package pilot

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Actual verifier code is exercised, with only the external LINE transport doubled.
func TestProductionVerifierRejectsInvalidClaims(t *testing.T) {
	f := setup(t)
	old := http.DefaultTransport
	defer func() { http.DefaultTransport = old }()
	cases := []struct {
		name          string
		status        int
		iss, aud, sub string
		expiry        int64
		valid         bool
	}{
		{"valid", 200, "https://access.line.me", "test-channel", "verified-line-sub", f.clock.Load() + 60, true},
		{"wrong audience", 200, "https://access.line.me", "another-channel", "subject", f.clock.Load() + 60, false},
		{"wrong issuer", 200, "https://evil.example", "test-channel", "subject", f.clock.Load() + 60, false},
		{"expired", 200, "https://access.line.me", "test-channel", "subject", f.clock.Load(), false},
		{"missing subject", 200, "https://access.line.me", "test-channel", "", f.clock.Load() + 60, false},
		{"invalid signature rejected by LINE", 400, "", "", "", 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = transportFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method != "POST" || r.URL.String() != "https://api.line.me/oauth2/v2.1/verify" {
					t.Fatalf("unexpected verifier endpoint %s", r.URL)
				}
				r.ParseForm()
				if r.Form.Get("client_id") != "test-channel" || r.Form.Get("id_token") != "synthetic-id-token" {
					t.Fatal("wrong verification parameters")
				}
				raw, _ := json.Marshal(map[string]any{"iss": tc.iss, "aud": tc.aud, "sub": tc.sub, "exp": tc.expiry})
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(string(raw))), Header: make(http.Header)}, nil
			})
			subject, err := f.s.verifyLine(context.Background(), "synthetic-id-token")
			if tc.valid {
				if err != nil || subject != "verified-line-sub" {
					t.Fatal("valid verification rejected", err)
				}
			} else if err == nil {
				t.Fatal("invalid LINE identity accepted")
			}
		})
	}
}

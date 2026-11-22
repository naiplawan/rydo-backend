package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthEndpoints(t *testing.T) {
	for _, tc := range []struct {
		name      string
		path      string
		pingError error
		want      int
	}{
		{"liveness independent of database", "/health/live", errors.New("connection failed"), http.StatusOK},
		{"ready with database", "/health/ready", nil, http.StatusOK},
		{"unready without database", "/health/ready", errors.New("connection failed"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := newRouter(func(context.Context) error { return tc.pingError })
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if response.Code != tc.want {
				t.Fatalf("GET %s returned %d, want %d", tc.path, response.Code, tc.want)
			}
		})
	}
}

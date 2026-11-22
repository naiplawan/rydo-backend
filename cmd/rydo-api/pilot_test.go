package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"rydo-backend/internal/pilot"
	"testing"
)

// An omitted API mount would return 404 instead of the protected API's 401.
func TestPilotRouteRequiresAuthentication(t *testing.T) {
	api, err := pilot.New(nil, pilot.Config{Origin: "https://pilot.example.test", JWTKey: "test-access-key-01234567890123456789", PreviewKey: "test-preview-key-01234567890123456789", LineChannelID: "test-channel"})
	if err != nil {
		t.Fatal(err)
	}
	router := newRouter(func(context.Context) error { return nil }, api)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("GET /api/v1/me returned %d; want protected-route 401", response.Code)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("private response must be no-store")
	}
}

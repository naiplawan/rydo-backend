package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSwaggerUI(t *testing.T) {
	router := newRouter(func(context.Context) error { return nil })
	for _, path := range []string{"/swagger/index.html", "/swagger/swagger-ui.css", "/swagger/swagger-ui-bundle.js"} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("GET %s returned %d, want 200", path, response.Code)
			}
			if path == "/swagger/index.html" && !strings.Contains(response.Body.String(), "./openapi.json") {
				t.Fatal("Swagger UI does not reference the local OpenAPI document")
			}
		})
	}
}

func TestOpenAPIDescribesHealthResponses(t *testing.T) {
	router := newRouter(func(context.Context) error { return nil })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/swagger/openapi.json", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("OpenAPI endpoint returned %d, want 200", response.Code)
	}
	var spec struct {
		OpenAPI string
		Paths   map[string]map[string]struct {
			Responses map[string]json.RawMessage
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(spec.OpenAPI, "3.") {
		t.Fatalf("spec version %q is not OpenAPI 3", spec.OpenAPI)
	}
	for _, item := range []struct{ path, status string }{
		{"/health/live", "200"},
		{"/health/ready", "200"},
		{"/health/ready", "503"},
	} {
		if _, ok := spec.Paths[item.path]["get"].Responses[item.status]; !ok {
			t.Errorf("GET %s is missing response %s", item.path, item.status)
		}
	}
}

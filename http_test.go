package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewApp(nil).Routes()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

func TestInvalidPathIDsReturnBadRequest(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{name: "get user", method: http.MethodGet, path: "/api/users/abc"},
		{name: "get service", method: http.MethodGet, path: "/api/services/abc"},
		{name: "get exchange", method: http.MethodGet, path: "/api/exchanges/abc"},
		{name: "get stats", method: http.MethodGet, path: "/api/users/abc/stats"},
	}

	handler := NewApp(nil).Routes()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, recorder.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestProtectedEndpointsRequireUserID(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "update user", method: http.MethodPut, path: "/api/users/1", body: `{"pseudo":"alice"}`},
		{name: "replace skills", method: http.MethodPut, path: "/api/users/1/skills", body: `[]`},
		{name: "create service", method: http.MethodPost, path: "/api/services", body: `{"titre":"Cours Go"}`},
		{name: "create exchange", method: http.MethodPost, path: "/api/exchanges", body: `{"service_id":1}`},
		{name: "list exchanges", method: http.MethodGet, path: "/api/exchanges"},
	}

	handler := NewApp(nil).Routes()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf("%s %s status = %d, want %d", tt.method, tt.path, recorder.Code, http.StatusForbidden)
			}
		})
	}
}

func TestInvalidUserIDHeaderReturnsBadRequest(t *testing.T) {
	handler := NewApp(nil).Routes()
	request := httptest.NewRequest(http.MethodGet, "/api/exchanges", nil)
	request.Header.Set("X-UserID", "not-an-int")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

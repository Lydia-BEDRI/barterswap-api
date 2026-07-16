package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSMiddlewareHandlesPreflight(t *testing.T) {
	handler := corsMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called for preflight requests")
	}))

	request := httptest.NewRequest(http.MethodOptions, "/api/users", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Fatalf("expected CORS origin '*', got %q", got)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, X-UserID" {
		t.Fatalf("expected CORS headers to include X-UserID, got %q", got)
	}
}

func TestAuthMiddlewareRejectsInvalidUserID(t *testing.T) {
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("next handler should not be called with invalid X-UserID")
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	request.Header.Set("X-UserID", "abc")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
}

func TestAuthMiddlewareStoresUserIDInContext(t *testing.T) {
	handler := authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, err := authenticatedUserID(r)
		if err != nil {
			t.Fatalf("expected authenticated user id, got error %v", err)
		}
		if userID != 42 {
			t.Fatalf("expected user id 42, got %d", userID)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/services", nil)
	request.Header.Set("X-UserID", "42")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestRecoveryMiddlewareReturnsInternalServerError(t *testing.T) {
	handler := recoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	request := httptest.NewRequest(http.MethodGet, "/panic", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
}

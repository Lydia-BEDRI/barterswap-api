package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserStatsRejectsInvalidID(t *testing.T) {
	handler := NewApp(nil).Routes()
	request := httptest.NewRequest(http.MethodGet, "/api/users/invalid/stats", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("stats status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

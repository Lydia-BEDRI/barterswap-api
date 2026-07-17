package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// App groups the HTTP routes and the storage layer.
type App struct {
	store *Store
}

// NewApp creates an application using the provided store.
func NewApp(store *Store) *App {
	return &App{store: store}
}

// Routes returns the HTTP handler with all API routes and middlewares.
func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.handleHealth)
	a.registerUserRoutes(mux)
	a.registerServiceRoutes(mux)
	a.registerExchangeRoutes(mux)
	a.registerReviewRoutes(mux)
	a.registerStatsRoutes(mux)
	return chain(
		mux,
		recoveryMiddleware,
		loggingMiddleware,
		corsMiddleware,
		authMiddleware,
	)
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("%w: invalid JSON body", ErrInvalidInput)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, ErrInvalidUserID):
		status = http.StatusBadRequest
	case errors.Is(err, ErrDuplicateValue):
		status = http.StatusConflict
	case errors.Is(err, ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, ErrInsufficientCredits):
		status = http.StatusBadRequest
	case errors.Is(err, ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
	}

	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		writeError(w, fmt.Errorf("%w: invalid id", ErrInvalidInput))
		return 0, false
	}
	return id, true
}

func sameAuthenticatedUser(r *http.Request, userID int) bool {
	authenticatedID, err := authenticatedUserID(r)
	return err == nil && authenticatedID == userID
}

func authenticatedUserID(r *http.Request) (int, error) {
	if id, ok := r.Context().Value(authenticatedUserIDKey{}).(int); ok {
		return id, nil
	}

	header := r.Header.Get("X-UserID")
	if header == "" {
		return 0, ErrForbidden
	}

	id, err := strconv.Atoi(header)
	if err != nil || id <= 0 {
		return 0, ErrInvalidUserID
	}

	return id, nil
}

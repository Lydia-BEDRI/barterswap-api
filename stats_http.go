package main

import "net/http"

func (a *App) registerStatsRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/users/{id}/stats", a.handleGetUserStats)
}

func (a *App) handleGetUserStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r)
	if !ok {
		return
	}

	stats, err := a.store.GetUserStats(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

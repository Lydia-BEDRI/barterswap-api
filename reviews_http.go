package main

import "net/http"

func (a *App) registerReviewRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/exchanges/{id}/review", a.handleCreateReview)
	mux.HandleFunc("GET /api/users/{id}/reviews", a.handleListUserReviews)
	mux.HandleFunc("GET /api/services/{id}/reviews", a.handleListServiceReviews)
}

func (a *App) handleCreateReview(w http.ResponseWriter, r *http.Request) {
	exchangeID, ok := pathID(w, r)
	if !ok {
		return
	}
	authorID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	var input CreateReviewInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}

	review, err := a.store.CreateReview(r.Context(), exchangeID, authorID, input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, review)
}

func (a *App) handleListUserReviews(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r)
	if !ok {
		return
	}

	reviews, err := a.store.ListUserReviews(r.Context(), userID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (a *App) handleListServiceReviews(w http.ResponseWriter, r *http.Request) {
	serviceID, ok := pathID(w, r)
	if !ok {
		return
	}

	reviews, err := a.store.ListServiceReviews(r.Context(), serviceID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

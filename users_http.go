package main

import "net/http"

func (a *App) registerUserRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/users", a.handleCreateUser)
	mux.HandleFunc("GET /api/users/{id}", a.handleGetUser)
	mux.HandleFunc("PUT /api/users/{id}", a.handleUpdateUser)
	mux.HandleFunc("GET /api/users/{id}/skills", a.handleGetUserSkills)
	mux.HandleFunc("PUT /api/users/{id}/skills", a.handleReplaceUserSkills)
}

func (a *App) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var input CreateUserInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}

	user, err := a.store.CreateUser(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (a *App) handleGetUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	user, err := a.store.GetUser(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if !sameAuthenticatedUser(r, id) {
		writeError(w, ErrForbidden)
		return
	}

	var input UpdateUserInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}

	user, err := a.store.UpdateUser(r.Context(), id, input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (a *App) handleGetUserSkills(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	skills, err := a.store.GetUserSkills(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, skills)
}

func (a *App) handleReplaceUserSkills(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if !sameAuthenticatedUser(r, id) {
		writeError(w, ErrForbidden)
		return
	}

	var skills []Skill
	if err := readJSON(r, &skills); err != nil {
		writeError(w, err)
		return
	}

	updatedSkills, err := a.store.ReplaceUserSkills(r.Context(), id, skills)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updatedSkills)
}

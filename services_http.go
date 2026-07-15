package main

import "net/http"

func (a *App) registerServiceRoutes(mux *http.ServeMux) {
	register := func(prefix string) {
		mux.HandleFunc("POST "+prefix, a.handleCreateService)
		mux.HandleFunc("GET "+prefix, a.handleListServices)
		mux.HandleFunc("GET "+prefix+"/{id}", a.handleGetService)
		mux.HandleFunc("PUT "+prefix+"/{id}", a.handleUpdateService)
		mux.HandleFunc("DELETE "+prefix+"/{id}", a.handleDeleteService)
	}

	register("/api/services")
	register("/services")
}

func (a *App) handleCreateService(w http.ResponseWriter, r *http.Request) {
	providerID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	var input CreateServiceInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}

	service, err := a.store.CreateService(r.Context(), providerID, input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, service)
}

func (a *App) handleListServices(w http.ResponseWriter, r *http.Request) {
	filter := ServiceFilter{
		Categorie: r.URL.Query().Get("categorie"),
		Ville:     r.URL.Query().Get("ville"),
		Search:    r.URL.Query().Get("search"),
	}

	services, err := a.store.ListServices(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, services)
}

func (a *App) handleGetService(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	service, err := a.store.GetService(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, service)
}

func (a *App) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	userID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	var input UpdateServiceInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}

	service, err := a.store.UpdateService(r.Context(), id, userID, input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, service)
}

func (a *App) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	userID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	if err := a.store.DeleteService(r.Context(), id, userID); err != nil {
		writeError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

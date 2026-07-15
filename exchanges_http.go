package main

import (
	"context"
	"net/http"
)

func (a *App) registerExchangeRoutes(mux *http.ServeMux) {
	register := func(prefix string) {
		mux.HandleFunc("POST "+prefix, a.handleCreateExchange)
		mux.HandleFunc("GET "+prefix, a.handleListExchanges)
		mux.HandleFunc("GET "+prefix+"/{id}", a.handleGetExchange)
		mux.HandleFunc("PUT "+prefix+"/{id}/accept", a.handleAcceptExchange)
		mux.HandleFunc("PUT "+prefix+"/{id}/reject", a.handleRejectExchange)
		mux.HandleFunc("PUT "+prefix+"/{id}/complete", a.handleCompleteExchange)
		mux.HandleFunc("PUT "+prefix+"/{id}/cancel", a.handleCancelExchange)
	}

	register("/api/exchanges")
	register("/exchanges")
}

func (a *App) handleCreateExchange(w http.ResponseWriter, r *http.Request) {
	requesterID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	var input CreateExchangeInput
	if err := readJSON(r, &input); err != nil {
		writeError(w, err)
		return
	}

	exchange, err := a.store.CreateExchange(r.Context(), requesterID, input)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, exchange)
}

func (a *App) handleListExchanges(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	filter := ExchangeFilter{Status: r.URL.Query().Get("status")}
	exchanges, err := a.store.ListExchanges(r.Context(), userID, filter)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, exchanges)
}

func (a *App) handleGetExchange(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	userID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	exchange, err := a.store.GetExchangeForUser(r.Context(), id, userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, exchange)
}

func (a *App) handleAcceptExchange(w http.ResponseWriter, r *http.Request) {
	a.handleExchangeAction(w, r, a.store.AcceptExchange)
}

func (a *App) handleRejectExchange(w http.ResponseWriter, r *http.Request) {
	a.handleExchangeAction(w, r, a.store.RejectExchange)
}

func (a *App) handleCompleteExchange(w http.ResponseWriter, r *http.Request) {
	a.handleExchangeAction(w, r, a.store.CompleteExchange)
}

func (a *App) handleCancelExchange(w http.ResponseWriter, r *http.Request) {
	a.handleExchangeAction(w, r, a.store.CancelExchange)
}

func (a *App) handleExchangeAction(
	w http.ResponseWriter,
	r *http.Request,
	action func(ctx context.Context, exchangeID int, userID int) (Exchange, error),
) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}

	userID, err := authenticatedUserID(r)
	if err != nil {
		writeError(w, err)
		return
	}

	exchange, err := action(r.Context(), id, userID)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, exchange)
}

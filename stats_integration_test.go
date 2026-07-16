package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestUserStatsEndpointIntegration(t *testing.T) {
	store, db := integrationStore(t)
	requesterID := insertIntegrationUser(t, db, "stats-requester", 3)
	providerID := insertIntegrationUser(t, db, "stats-provider", 17)
	emptyUserID := insertIntegrationUser(t, db, "stats-empty", 10)
	t.Cleanup(func() {
		cleanupIntegrationUsers(t, db, requesterID, providerID, emptyUserID)
	})

	activeServiceID := insertIntegrationService(t, db, providerID, 4)
	inactiveServiceID := insertIntegrationService(t, db, providerID, 3)
	if _, err := db.Exec(`UPDATE services SET actif = FALSE WHERE id = ?`, inactiveServiceID); err != nil {
		t.Fatalf("disable integration service: %v", err)
	}

	firstExchangeID := insertReviewExchange(
		t, db, activeServiceID, requesterID, providerID, exchangeStatusCompleted,
	)
	secondExchangeID := insertReviewExchange(
		t, db, inactiveServiceID, requesterID, providerID, exchangeStatusCompleted,
	)
	insertStatsCreditTransaction(t, db, requesterID, firstExchangeID, -4, creditTransactionSpend)
	insertStatsCreditTransaction(t, db, providerID, firstExchangeID, 4, creditTransactionEarn)
	insertStatsCreditTransaction(t, db, requesterID, secondExchangeID, -3, creditTransactionSpend)
	insertStatsCreditTransaction(t, db, providerID, secondExchangeID, 3, creditTransactionEarn)
	insertStatsReview(t, db, firstExchangeID, requesterID, providerID, 5)
	insertStatsReview(t, db, secondExchangeID, requesterID, providerID, 3)
	insertStatsReview(t, db, firstExchangeID, providerID, requesterID, 4)

	handler := NewApp(store).Routes()
	providerStats := getStatsResponse(t, handler, providerID, http.StatusOK)
	assertUserStats(t, providerStats, UserStats{
		UserID:            providerID,
		ServicesActifs:    1,
		EchangesCompletes: 2,
		CreditBalance:     17,
		NoteMoyenne:       4,
		NbAvis:            2,
		TotalGagne:        7,
		TotalDepense:      0,
	})

	requesterStats := getStatsResponse(t, handler, requesterID, http.StatusOK)
	assertUserStats(t, requesterStats, UserStats{
		UserID:            requesterID,
		ServicesActifs:    0,
		EchangesCompletes: 2,
		CreditBalance:     3,
		NoteMoyenne:       4,
		NbAvis:            1,
		TotalGagne:        0,
		TotalDepense:      7,
	})

	emptyStats := getStatsResponse(t, handler, emptyUserID, http.StatusOK)
	assertUserStats(t, emptyStats, UserStats{UserID: emptyUserID, CreditBalance: 10})
	getStatsResponse(t, handler, 999999999, http.StatusNotFound)
}

func insertStatsCreditTransaction(
	t *testing.T,
	db *sql.DB,
	userID int,
	exchangeID int,
	amount int,
	transactionType string,
) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
		VALUES (?, ?, ?, ?)
	`, userID, exchangeID, amount, transactionType); err != nil {
		t.Fatalf("insert stats credit transaction: %v", err)
	}
}

func insertStatsReview(
	t *testing.T,
	db *sql.DB,
	exchangeID int,
	authorID int,
	targetID int,
	note int,
) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO reviews (exchange_id, author_id, target_id, note, commentaire)
		VALUES (?, ?, ?, ?, NULL)
	`, exchangeID, authorID, targetID, note); err != nil {
		t.Fatalf("insert stats review: %v", err)
	}
}

func getStatsResponse(
	t *testing.T,
	handler http.Handler,
	userID int,
	wantStatus int,
) UserStats {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/users/%d/stats", userID), nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != wantStatus {
		t.Fatalf("stats status = %d, want %d, body = %s", recorder.Code, wantStatus, recorder.Body.String())
	}
	if wantStatus != http.StatusOK {
		return UserStats{}
	}

	var stats UserStats
	if err := json.NewDecoder(recorder.Body).Decode(&stats); err != nil {
		t.Fatalf("decode stats response: %v", err)
	}
	return stats
}

func assertUserStats(t *testing.T, got UserStats, want UserStats) {
	t.Helper()
	if got != want {
		t.Fatalf("stats = %+v, want %+v", got, want)
	}
}

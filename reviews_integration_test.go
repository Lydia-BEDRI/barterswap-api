package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReviewEndpointsIntegration(t *testing.T) {
	store, db := integrationStore(t)
	requesterID := insertIntegrationUser(t, db, "review-requester", 10)
	providerID := insertIntegrationUser(t, db, "review-provider", 10)
	outsiderID := insertIntegrationUser(t, db, "review-outsider", 10)
	t.Cleanup(func() {
		cleanupIntegrationUsers(t, db, requesterID, providerID, outsiderID)
	})

	serviceID := insertIntegrationService(t, db, providerID, 4)
	completedExchangeID := insertReviewExchange(
		t, db, serviceID, requesterID, providerID, exchangeStatusCompleted,
	)
	pendingExchangeID := insertReviewExchange(
		t, db, serviceID, requesterID, providerID, exchangeStatusPending,
	)
	handler := NewApp(store).Routes()

	requesterReview := performReviewRequest(
		t,
		handler,
		http.MethodPost,
		fmt.Sprintf("/api/exchanges/%d/review", completedExchangeID),
		requesterID,
		`{"note":5,"commentaire":"  Très bon service  "}`,
	)
	if requesterReview.Code != http.StatusCreated {
		t.Fatalf("requester review status = %d, body = %s", requesterReview.Code, requesterReview.Body.String())
	}
	created := decodeReview(t, requesterReview)
	if created.AuthorID != requesterID || created.TargetID != providerID || created.Note != 5 {
		t.Fatalf("created requester review = %+v", created)
	}
	if created.Commentaire != "Très bon service" {
		t.Fatalf("created comment = %q, want trimmed comment", created.Commentaire)
	}

	providerReview := performReviewRequest(
		t,
		handler,
		http.MethodPost,
		fmt.Sprintf("/api/exchanges/%d/review", completedExchangeID),
		providerID,
		`{"note":4,"commentaire":"Échange sérieux"}`,
	)
	if providerReview.Code != http.StatusCreated {
		t.Fatalf("provider review status = %d, body = %s", providerReview.Code, providerReview.Body.String())
	}
	created = decodeReview(t, providerReview)
	if created.AuthorID != providerID || created.TargetID != requesterID || created.Note != 4 {
		t.Fatalf("created provider review = %+v", created)
	}

	duplicate := performReviewRequest(
		t,
		handler,
		http.MethodPost,
		fmt.Sprintf("/api/exchanges/%d/review", completedExchangeID),
		requesterID,
		`{"note":3}`,
	)
	if duplicate.Code != http.StatusBadRequest {
		t.Fatalf("duplicate review status = %d, want %d", duplicate.Code, http.StatusBadRequest)
	}

	outsider := performReviewRequest(
		t,
		handler,
		http.MethodPost,
		fmt.Sprintf("/api/exchanges/%d/review", completedExchangeID),
		outsiderID,
		`{"note":3}`,
	)
	if outsider.Code != http.StatusForbidden {
		t.Fatalf("outsider review status = %d, want %d", outsider.Code, http.StatusForbidden)
	}

	pending := performReviewRequest(
		t,
		handler,
		http.MethodPost,
		fmt.Sprintf("/api/exchanges/%d/review", pendingExchangeID),
		requesterID,
		`{"note":3}`,
	)
	if pending.Code != http.StatusBadRequest {
		t.Fatalf("pending review status = %d, want %d", pending.Code, http.StatusBadRequest)
	}

	assertReviewList(t, handler, fmt.Sprintf("/api/users/%d/reviews", providerID), 1, requesterID, providerID)
	assertReviewList(t, handler, fmt.Sprintf("/api/users/%d/reviews", requesterID), 1, providerID, requesterID)
	assertReviewList(t, handler, fmt.Sprintf("/api/users/%d/reviews", outsiderID), 0, 0, 0)

	// The provider's review targets the requester, so it is not a review of the service.
	assertReviewList(t, handler, fmt.Sprintf("/api/services/%d/reviews", serviceID), 1, requesterID, providerID)
}

func insertReviewExchange(
	t *testing.T,
	db *sql.DB,
	serviceID int,
	requesterID int,
	ownerID int,
	status string,
) int {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO exchanges (service_id, requester_id, owner_id, status)
		VALUES (?, ?, ?, ?)
	`, serviceID, requesterID, ownerID, status)
	if err != nil {
		t.Fatalf("insert review exchange: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("review exchange LastInsertId(): %v", err)
	}
	return int(id)
}

func performReviewRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	userID int,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID > 0 {
		request.Header.Set("X-UserID", fmt.Sprintf("%d", userID))
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeReview(t *testing.T, recorder *httptest.ResponseRecorder) Review {
	t.Helper()
	var review Review
	if err := json.NewDecoder(recorder.Body).Decode(&review); err != nil {
		t.Fatalf("decode review response: %v", err)
	}
	return review
}

func assertReviewList(
	t *testing.T,
	handler http.Handler,
	path string,
	wantLength int,
	wantAuthorID int,
	wantTargetID int,
) {
	t.Helper()
	recorder := performReviewRequest(t, handler, http.MethodGet, path, 0, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, body = %s", path, recorder.Code, recorder.Body.String())
	}

	var reviews []Review
	if err := json.NewDecoder(recorder.Body).Decode(&reviews); err != nil {
		t.Fatalf("decode reviews for %s: %v", path, err)
	}
	if len(reviews) != wantLength {
		t.Fatalf("GET %s returned %d reviews, want %d", path, len(reviews), wantLength)
	}
	if wantLength > 0 && (reviews[0].AuthorID != wantAuthorID || reviews[0].TargetID != wantTargetID) {
		t.Fatalf("GET %s returned %+v", path, reviews[0])
	}
}

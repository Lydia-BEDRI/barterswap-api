package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUserEndpointsIntegration(t *testing.T) {
	store, db := integrationStore(t)
	handler := NewApp(store).Routes()

	pseudo := fmt.Sprintf("api-user-%d", time.Now().UnixNano())
	create := performAPIRequest(
		t,
		handler,
		http.MethodPost,
		"/api/users",
		0,
		fmt.Sprintf(`{"pseudo":%q,"bio":"  Bonjour  ","ville":" Paris "}`, pseudo),
	)
	if create.Code != http.StatusCreated {
		t.Fatalf("create user status = %d, body = %s", create.Code, create.Body.String())
	}
	user := decodeAPIResponse[User](t, create)
	t.Cleanup(func() { cleanupIntegrationUsers(t, db, user.ID) })
	if user.CreditBalance != 10 {
		t.Fatalf("created user credit balance = %d, want 10", user.CreditBalance)
	}

	duplicate := performAPIRequest(
		t, handler, http.MethodPost, "/api/users", 0, fmt.Sprintf(`{"pseudo":%q}`, pseudo),
	)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate user status = %d, want %d", duplicate.Code, http.StatusConflict)
	}

	get := performAPIRequest(t, handler, http.MethodGet, fmt.Sprintf("/api/users/%d", user.ID), 0, "")
	if get.Code != http.StatusOK {
		t.Fatalf("get user status = %d, body = %s", get.Code, get.Body.String())
	}

	forbidden := performAPIRequest(
		t, handler, http.MethodPut, fmt.Sprintf("/api/users/%d", user.ID), 0, `{"pseudo":"forbidden"}`,
	)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("update without auth status = %d, want %d", forbidden.Code, http.StatusForbidden)
	}

	updatedPseudo := pseudo + "-updated"
	update := performAPIRequest(
		t,
		handler,
		http.MethodPut,
		fmt.Sprintf("/api/users/%d", user.ID),
		user.ID,
		fmt.Sprintf(`{"pseudo":%q,"bio":"Modifié","ville":"Lyon"}`, updatedPseudo),
	)
	if update.Code != http.StatusOK {
		t.Fatalf("update user status = %d, body = %s", update.Code, update.Body.String())
	}
	updated := decodeAPIResponse[User](t, update)
	if updated.Pseudo != updatedPseudo || updated.Ville != "Lyon" {
		t.Fatalf("updated user = %+v", updated)
	}

	replaceSkills := performAPIRequest(
		t,
		handler,
		http.MethodPut,
		fmt.Sprintf("/api/users/%d/skills", user.ID),
		user.ID,
		`[{"nom":"Go","niveau":"intermédiaire"},{"nom":"Cuisine","niveau":"débutant"}]`,
	)
	if replaceSkills.Code != http.StatusOK {
		t.Fatalf("replace skills status = %d, body = %s", replaceSkills.Code, replaceSkills.Body.String())
	}
	skills := decodeAPIResponse[[]Skill](t, replaceSkills)
	if len(skills) != 2 {
		t.Fatalf("skills length = %d, want 2", len(skills))
	}
}

func TestServiceEndpointsIntegration(t *testing.T) {
	store, db := integrationStore(t)
	handler := NewApp(store).Routes()

	providerID := insertIntegrationUser(t, db, "api-service-provider", 10)
	outsiderID := insertIntegrationUser(t, db, "api-service-outsider", 10)
	t.Cleanup(func() { cleanupIntegrationUsers(t, db, providerID, outsiderID) })
	insertIntegrationSkill(t, db, providerID, "Go", "expert")

	create := performAPIRequest(
		t,
		handler,
		http.MethodPost,
		"/api/services",
		providerID,
		`{"titre":" Cours Go ","description":"Bases","categorie":"Informatique","duree_minutes":90,"credits":4,"ville":"Paris"}`,
	)
	if create.Code != http.StatusCreated {
		t.Fatalf("create service status = %d, body = %s", create.Code, create.Body.String())
	}
	service := decodeAPIResponse[Service](t, create)
	if service.ProviderID != providerID || service.Titre != "Cours Go" || service.Credits != 4 {
		t.Fatalf("created service = %+v", service)
	}

	list := performAPIRequest(t, handler, http.MethodGet, "/api/services?categorie=Informatique&ville=Paris", 0, "")
	if list.Code != http.StatusOK {
		t.Fatalf("list services status = %d, body = %s", list.Code, list.Body.String())
	}
	services := decodeAPIResponse[[]Service](t, list)
	if len(services) == 0 {
		t.Fatalf("list services returned no service")
	}

	forbidden := performAPIRequest(
		t,
		handler,
		http.MethodPut,
		fmt.Sprintf("/api/services/%d", service.ID),
		outsiderID,
		`{"titre":"Vol","categorie":"Informatique","duree_minutes":60,"credits":2}`,
	)
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("outsider update status = %d, want %d", forbidden.Code, http.StatusForbidden)
	}

	deleteResponse := performAPIRequest(
		t, handler, http.MethodDelete, fmt.Sprintf("/api/services/%d", service.ID), providerID, "",
	)
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("delete service status = %d, body = %s", deleteResponse.Code, deleteResponse.Body.String())
	}

	getDeleted := performAPIRequest(t, handler, http.MethodGet, fmt.Sprintf("/api/services/%d", service.ID), 0, "")
	if getDeleted.Code != http.StatusNotFound {
		t.Fatalf("get deleted service status = %d, want %d", getDeleted.Code, http.StatusNotFound)
	}
}

func TestExchangeEndpointsIntegration(t *testing.T) {
	store, db := integrationStore(t)
	handler := NewApp(store).Routes()

	requesterID := insertIntegrationUser(t, db, "api-exchange-requester", 10)
	providerID := insertIntegrationUser(t, db, "api-exchange-provider", 10)
	outsiderID := insertIntegrationUser(t, db, "api-exchange-outsider", 10)
	t.Cleanup(func() { cleanupIntegrationUsers(t, db, requesterID, providerID, outsiderID) })

	serviceID := insertIntegrationService(t, db, providerID, 4)
	create := performAPIRequest(
		t,
		handler,
		http.MethodPost,
		"/api/exchanges",
		requesterID,
		fmt.Sprintf(`{"service_id":%d}`, serviceID),
	)
	if create.Code != http.StatusCreated {
		t.Fatalf("create exchange status = %d, body = %s", create.Code, create.Body.String())
	}
	exchange := decodeAPIResponse[Exchange](t, create)
	if exchange.Status != exchangeStatusPending {
		t.Fatalf("exchange status = %q, want %q", exchange.Status, exchangeStatusPending)
	}

	duplicate := performAPIRequest(
		t,
		handler,
		http.MethodPost,
		"/api/exchanges",
		requesterID,
		fmt.Sprintf(`{"service_id":%d}`, serviceID),
	)
	if duplicate.Code != http.StatusConflict {
		t.Fatalf("duplicate active exchange status = %d, want %d", duplicate.Code, http.StatusConflict)
	}

	outsiderAccept := performAPIRequest(
		t,
		handler,
		http.MethodPut,
		fmt.Sprintf("/api/exchanges/%d/accept", exchange.ID),
		outsiderID,
		"",
	)
	if outsiderAccept.Code != http.StatusForbidden {
		t.Fatalf("outsider accept status = %d, want %d", outsiderAccept.Code, http.StatusForbidden)
	}

	accept := performAPIRequest(
		t,
		handler,
		http.MethodPut,
		fmt.Sprintf("/api/exchanges/%d/accept", exchange.ID),
		providerID,
		"",
	)
	if accept.Code != http.StatusOK {
		t.Fatalf("accept exchange status = %d, body = %s", accept.Code, accept.Body.String())
	}
	accepted := decodeAPIResponse[Exchange](t, accept)
	if accepted.Status != exchangeStatusAccepted {
		t.Fatalf("accepted status = %q, want %q", accepted.Status, exchangeStatusAccepted)
	}

	complete := performAPIRequest(
		t,
		handler,
		http.MethodPut,
		fmt.Sprintf("/api/exchanges/%d/complete", exchange.ID),
		requesterID,
		"",
	)
	if complete.Code != http.StatusOK {
		t.Fatalf("complete exchange status = %d, body = %s", complete.Code, complete.Body.String())
	}
	completed := decodeAPIResponse[Exchange](t, complete)
	if completed.Status != exchangeStatusCompleted {
		t.Fatalf("completed status = %q, want %q", completed.Status, exchangeStatusCompleted)
	}
}

func insertIntegrationSkill(t *testing.T, db *sql.DB, userID int, nom string, niveau string) {
	t.Helper()
	if _, err := db.Exec(`
		INSERT INTO user_skills (user_id, nom, niveau)
		VALUES (?, ?, ?)
	`, userID, nom, niveau); err != nil {
		t.Fatalf("insert integration skill: %v", err)
	}
}

func performAPIRequest(
	t *testing.T,
	handler http.Handler,
	method string,
	path string,
	userID int,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	if userID > 0 {
		request.Header.Set("X-UserID", fmt.Sprintf("%d", userID))
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func decodeAPIResponse[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var payload T
	if err := json.NewDecoder(recorder.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return payload
}

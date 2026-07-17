package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestCreditLifecycleIntegration(t *testing.T) {
	store, db := integrationStore(t)
	requesterID := insertIntegrationUser(t, db, "requester", 10)
	providerID := insertIntegrationUser(t, db, "provider", 10)
	t.Cleanup(func() { cleanupIntegrationUsers(t, db, requesterID, providerID) })

	serviceID := insertIntegrationService(t, db, providerID, 4)
	exchange, err := store.CreateExchange(context.Background(), requesterID, CreateExchangeInput{ServiceID: serviceID})
	if err != nil {
		t.Fatalf("CreateExchange() error = %v", err)
	}
	if _, err := store.AcceptExchange(context.Background(), exchange.ID, providerID); err != nil {
		t.Fatalf("AcceptExchange() error = %v", err)
	}
	assertCreditBalance(t, store, requesterID, 6)
	assertJournalEntry(t, store, requesterID, exchange.ID, creditTransactionSpend, -4)

	if _, err := store.CompleteExchange(context.Background(), exchange.ID, requesterID); err != nil {
		t.Fatalf("requester CompleteExchange() error = %v", err)
	}
	assertCreditBalance(t, store, requesterID, 6)
	assertCreditBalance(t, store, providerID, 14)
	assertJournalEntry(t, store, providerID, exchange.ID, creditTransactionEarn, 4)

	refundedServiceID := insertIntegrationService(t, db, providerID, 3)
	refundedExchange, err := store.CreateExchange(
		context.Background(), requesterID, CreateExchangeInput{ServiceID: refundedServiceID},
	)
	if err != nil {
		t.Fatalf("CreateExchange(refund) error = %v", err)
	}
	if _, err := store.AcceptExchange(context.Background(), refundedExchange.ID, providerID); err != nil {
		t.Fatalf("AcceptExchange(refund) error = %v", err)
	}
	assertCreditBalance(t, store, requesterID, 3)

	if _, err := store.CancelExchange(context.Background(), refundedExchange.ID, providerID); err != nil {
		t.Fatalf("CancelExchange() error = %v", err)
	}
	assertCreditBalance(t, store, requesterID, 6)
	assertJournalEntry(t, store, requesterID, refundedExchange.ID, creditTransactionRefund, 3)
}

func TestConcurrentCreditBlocksIntegration(t *testing.T) {
	store, db := integrationStore(t)
	requesterID := insertIntegrationUser(t, db, "concurrent-requester", 10)
	providerOneID := insertIntegrationUser(t, db, "concurrent-provider-one", 10)
	providerTwoID := insertIntegrationUser(t, db, "concurrent-provider-two", 10)
	t.Cleanup(func() {
		cleanupIntegrationUsers(t, db, requesterID, providerOneID, providerTwoID)
	})

	serviceOneID := insertIntegrationService(t, db, providerOneID, 7)
	serviceTwoID := insertIntegrationService(t, db, providerTwoID, 7)
	exchangeOne, err := store.CreateExchange(context.Background(), requesterID, CreateExchangeInput{ServiceID: serviceOneID})
	if err != nil {
		t.Fatalf("CreateExchange(one) error = %v", err)
	}
	exchangeTwo, err := store.CreateExchange(context.Background(), requesterID, CreateExchangeInput{ServiceID: serviceTwoID})
	if err != nil {
		t.Fatalf("CreateExchange(two) error = %v", err)
	}

	type acceptResult struct {
		exchangeID int
		err        error
	}
	results := make(chan acceptResult, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, attempt := range []struct {
		exchangeID int
		providerID int
	}{
		{exchangeID: exchangeOne.ID, providerID: providerOneID},
		{exchangeID: exchangeTwo.ID, providerID: providerTwoID},
	} {
		wg.Add(1)
		go func(exchangeID int, providerID int) {
			defer wg.Done()
			<-start
			_, err := store.AcceptExchange(context.Background(), exchangeID, providerID)
			results <- acceptResult{exchangeID: exchangeID, err: err}
		}(attempt.exchangeID, attempt.providerID)
	}
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	insufficient := 0
	for result := range results {
		switch {
		case result.err == nil:
			successes++
		case errors.Is(result.err, ErrInsufficientCredits):
			insufficient++
		default:
			t.Fatalf("AcceptExchange(%d) error = %v", result.exchangeID, result.err)
		}
	}
	if successes != 1 || insufficient != 1 {
		t.Fatalf("concurrent accepts: successes=%d insufficient=%d, want 1 and 1", successes, insufficient)
	}
	assertCreditBalance(t, store, requesterID, 3)

	journal, err := store.ListCreditTransactions(
		context.Background(), requesterID, CreditTransactionFilter{Type: creditTransactionSpend},
	)
	if err != nil {
		t.Fatalf("ListCreditTransactions() error = %v", err)
	}
	if len(journal) != 1 {
		t.Fatalf("block journal length = %d, want 1", len(journal))
	}
}

func integrationStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN is not set")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("database ping error = %v", err)
	}
	return NewStore(db), db
}

func insertIntegrationUser(t *testing.T, db *sql.DB, label string, balance int) int {
	t.Helper()
	pseudo := fmt.Sprintf("credit-test-%s-%d", label, time.Now().UnixNano())
	result, err := db.Exec(`
		INSERT INTO users (pseudo, bio, ville, credit_balance)
		VALUES (?, NULL, NULL, ?)
	`, pseudo, balance)
	if err != nil {
		t.Fatalf("insert integration user: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("integration user LastInsertId(): %v", err)
	}
	return int(id)
}

func insertIntegrationService(t *testing.T, db *sql.DB, providerID int, credits int) int {
	t.Helper()
	result, err := db.Exec(`
		INSERT INTO services (
			provider_id, titre, description, categorie, duree_minutes, credits, ville, actif
		) VALUES (?, ?, NULL, 'Autre', 60, ?, NULL, TRUE)
	`, providerID, fmt.Sprintf("Credit test %d", time.Now().UnixNano()), credits)
	if err != nil {
		t.Fatalf("insert integration service: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("integration service LastInsertId(): %v", err)
	}
	return int(id)
}

func cleanupIntegrationUsers(t *testing.T, db *sql.DB, userIDs ...int) {
	t.Helper()
	if len(userIDs) == 0 {
		return
	}

	placeholders := "?"
	args := []any{userIDs[0]}
	for _, userID := range userIDs[1:] {
		placeholders += ", ?"
		args = append(args, userID)
	}
	queries := []string{
		"DELETE FROM credit_transactions WHERE user_id IN (" + placeholders + ")",
		"DELETE FROM exchanges WHERE requester_id IN (" + placeholders + ") OR owner_id IN (" + placeholders + ")",
		"DELETE FROM services WHERE provider_id IN (" + placeholders + ")",
		"DELETE FROM users WHERE id IN (" + placeholders + ")",
	}
	for index, query := range queries {
		queryArgs := args
		if index == 1 {
			queryArgs = append(append([]any{}, args...), args...)
		}
		if _, err := db.Exec(query, queryArgs...); err != nil {
			t.Errorf("cleanup integration data: %v", err)
		}
	}
}

func assertCreditBalance(
	t *testing.T,
	store *Store,
	userID int,
	want int,
) {
	t.Helper()
	user, err := store.getUserWithoutSkills(context.Background(), userID)
	if err != nil {
		t.Fatalf("getUserWithoutSkills() error = %v", err)
	}
	if user.CreditBalance != want {
		t.Fatalf("credit balance = %d, want %d", user.CreditBalance, want)
	}
}

func assertJournalEntry(
	t *testing.T,
	store *Store,
	userID int,
	exchangeID int,
	transactionType string,
	wantAmount int,
) {
	t.Helper()
	journal, err := store.ListCreditTransactions(context.Background(), userID, CreditTransactionFilter{
		Type:       transactionType,
		ExchangeID: exchangeID,
	})
	if err != nil {
		t.Fatalf("ListCreditTransactions() error = %v", err)
	}
	if len(journal) != 1 || journal[0].Montant != wantAmount {
		t.Fatalf("journal = %+v, want one %s transaction for %d", journal, transactionType, wantAmount)
	}
}

package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNextCreditBalance(t *testing.T) {
	tests := []struct {
		name      string
		balance   int
		delta     int
		want      int
		wantError error
	}{
		{name: "spend", balance: 10, delta: -4, want: 6},
		{name: "earn", balance: 10, delta: 4, want: 14},
		{name: "refund", balance: 6, delta: 4, want: 10},
		{name: "exact balance", balance: 4, delta: -4, want: 0},
		{name: "insufficient credits", balance: 3, delta: -4, wantError: ErrInsufficientCredits},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := nextCreditBalance(tt.balance, tt.delta)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("nextCreditBalance() error = %v, want %v", err, tt.wantError)
			}
			if got != tt.want {
				t.Fatalf("nextCreditBalance() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValidateCreditTransaction(t *testing.T) {
	tests := []struct {
		name            string
		amount          int
		transactionType string
		wantError       bool
	}{
		{name: "spend", amount: -5, transactionType: creditTransactionSpend},
		{name: "earn", amount: 5, transactionType: creditTransactionEarn},
		{name: "refund", amount: 5, transactionType: creditTransactionRefund},
		{name: "positive spend", amount: 5, transactionType: creditTransactionSpend, wantError: true},
		{name: "negative earn", amount: -5, transactionType: creditTransactionEarn, wantError: true},
		{name: "zero refund", amount: 0, transactionType: creditTransactionRefund, wantError: true},
		{name: "unknown type", amount: 5, transactionType: "block", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCreditTransaction(tt.amount, tt.transactionType)
			if (err != nil) != tt.wantError {
				t.Fatalf("validateCreditTransaction() error = %v, wantError %t", err, tt.wantError)
			}
		})
	}
}

func TestInsufficientCreditsHTTPStatus(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, ErrInsufficientCredits)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("writeError() status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

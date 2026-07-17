package main

import "testing"

func TestValidExchangeStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{status: exchangeStatusPending, want: true},
		{status: exchangeStatusAccepted, want: true},
		{status: exchangeStatusRejected, want: true},
		{status: exchangeStatusCancelled, want: true},
		{status: exchangeStatusCompleted, want: true},
		{status: "draft", want: false},
		{status: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := validExchangeStatus(tt.status); got != tt.want {
				t.Fatalf("validExchangeStatus(%q) = %t, want %t", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsExchangeParticipant(t *testing.T) {
	exchange := Exchange{RequesterID: 10, OwnerID: 20}

	tests := []struct {
		name   string
		userID int
		want   bool
	}{
		{name: "requester", userID: 10, want: true},
		{name: "owner", userID: 20, want: true},
		{name: "outsider", userID: 30, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isExchangeParticipant(exchange, tt.userID); got != tt.want {
				t.Fatalf("isExchangeParticipant() = %t, want %t", got, tt.want)
			}
		})
	}
}

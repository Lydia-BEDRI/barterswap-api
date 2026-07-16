package main

import "testing"

func TestValidateReviewInput(t *testing.T) {
	tests := []struct {
		name      string
		note      int
		wantError bool
	}{
		{name: "minimum", note: 1},
		{name: "maximum", note: 5},
		{name: "below minimum", note: 0, wantError: true},
		{name: "above maximum", note: 6, wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateReviewInput(CreateReviewInput{Note: tt.note})
			if (err != nil) != tt.wantError {
				t.Fatalf("validateReviewInput() error = %v, wantError %t", err, tt.wantError)
			}
		})
	}
}

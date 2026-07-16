package main

import (
	"errors"
	"testing"
)

func TestValidateServiceInput(t *testing.T) {
	tests := []struct {
		name      string
		titre     string
		categorie string
		duration  int
		credits   int
		wantError error
	}{
		{name: "valid", titre: "Aide Go", categorie: "Informatique", duration: 60, credits: 3},
		{name: "missing title", categorie: "Informatique", duration: 60, credits: 3, wantError: ErrInvalidInput},
		{name: "invalid category", titre: "Aide Go", categorie: "Magie", duration: 60, credits: 3, wantError: ErrInvalidInput},
		{name: "zero duration", titre: "Aide Go", categorie: "Informatique", duration: 0, credits: 3, wantError: ErrInvalidInput},
		{name: "zero credits", titre: "Aide Go", categorie: "Informatique", duration: 60, credits: 0, wantError: ErrInvalidInput},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateServiceInput(tt.titre, tt.categorie, tt.duration, tt.credits)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("validateServiceInput() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestValidServiceCategory(t *testing.T) {
	tests := []struct {
		category string
		want     bool
	}{
		{category: "Informatique", want: true},
		{category: "Jardinage", want: true},
		{category: "Autre", want: true},
		{category: "Déménagement", want: true},
		{category: "Finance", want: false},
		{category: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			if got := validServiceCategory(tt.category); got != tt.want {
				t.Fatalf("validServiceCategory(%q) = %t, want %t", tt.category, got, tt.want)
			}
		})
	}
}

func TestNormalizeServiceInput(t *testing.T) {
	input := normalizeCreateServiceInput(CreateServiceInput{
		Titre:       "  Cours Go  ",
		Description: "  Bases du langage  ",
		Categorie:   "  Informatique  ",
		Ville:       "  Paris  ",
	})

	if input.Titre != "Cours Go" {
		t.Fatalf("Titre = %q, want trimmed value", input.Titre)
	}
	if input.Description != "Bases du langage" {
		t.Fatalf("Description = %q, want trimmed value", input.Description)
	}
	if input.Categorie != "Informatique" {
		t.Fatalf("Categorie = %q, want trimmed value", input.Categorie)
	}
	if input.Ville != "Paris" {
		t.Fatalf("Ville = %q, want trimmed value", input.Ville)
	}
}

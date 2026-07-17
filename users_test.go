package main

import (
	"errors"
	"testing"
)

func TestValidateSkills(t *testing.T) {
	tests := []struct {
		name      string
		skills    []Skill
		wantError error
	}{
		{
			name: "valid skills",
			skills: []Skill{
				{Nom: "Go", Niveau: "intermédiaire"},
				{Nom: "Cuisine", Niveau: "débutant"},
			},
		},
		{
			name:      "empty skill name",
			skills:    []Skill{{Nom: " ", Niveau: "expert"}},
			wantError: ErrInvalidInput,
		},
		{
			name:      "invalid skill level",
			skills:    []Skill{{Nom: "Go", Niveau: "maître"}},
			wantError: ErrInvalidInput,
		},
		{
			name: "duplicate skill case insensitive",
			skills: []Skill{
				{Nom: "Go", Niveau: "expert"},
				{Nom: "go", Niveau: "débutant"},
			},
			wantError: ErrInvalidInput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSkills(tt.skills)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("validateSkills() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestValidSkillLevel(t *testing.T) {
	tests := []struct {
		level string
		want  bool
	}{
		{level: "débutant", want: true},
		{level: "intermédiaire", want: true},
		{level: "expert", want: true},
		{level: "novice", want: false},
		{level: "", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			if got := validSkillLevel(tt.level); got != tt.want {
				t.Fatalf("validSkillLevel(%q) = %t, want %t", tt.level, got, tt.want)
			}
		})
	}
}

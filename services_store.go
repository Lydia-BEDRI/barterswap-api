package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type CreateServiceInput struct {
	Titre        string `json:"titre"`
	Description  string `json:"description,omitempty"`
	Categorie    string `json:"categorie"`
	DureeMinutes int    `json:"duree_minutes"`
	Credits      int    `json:"credits"`
	Ville        string `json:"ville,omitempty"`
}

type UpdateServiceInput struct {
	Titre        string `json:"titre"`
	Description  string `json:"description,omitempty"`
	Categorie    string `json:"categorie"`
	DureeMinutes int    `json:"duree_minutes"`
	Credits      int    `json:"credits"`
	Ville        string `json:"ville,omitempty"`
	Actif        *bool  `json:"actif,omitempty"`
}

type ServiceFilter struct {
	Categorie string
	Ville     string
	Search    string
}

func (s *Store) CreateService(ctx context.Context, providerID int, input CreateServiceInput) (Service, error) {
	input = normalizeCreateServiceInput(input)
	if err := validateServiceInput(input.Titre, input.Categorie, input.DureeMinutes, input.Credits); err != nil {
		return Service{}, err
	}
	if err := s.validateServiceProvider(ctx, providerID); err != nil {
		return Service{}, err
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO services (provider_id, titre, description, categorie, duree_minutes, credits, ville, actif)
		VALUES (?, ?, ?, ?, ?, ?, ?, TRUE)
	`, providerID, input.Titre, nullableString(input.Description), input.Categorie, input.DureeMinutes, input.Credits, nullableString(input.Ville))
	if err != nil {
		return Service{}, fmt.Errorf("insert service: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Service{}, fmt.Errorf("read inserted service id: %w", err)
	}

	return s.GetService(ctx, int(id))
}

func (s *Store) ListServices(ctx context.Context, filter ServiceFilter) ([]Service, error) {
	filter.Categorie = strings.TrimSpace(filter.Categorie)
	filter.Ville = strings.TrimSpace(filter.Ville)
	filter.Search = strings.TrimSpace(filter.Search)

	if filter.Categorie != "" && !validServiceCategory(filter.Categorie) {
		return nil, fmt.Errorf("%w: invalid service categorie", ErrInvalidInput)
	}

	query := `
		SELECT id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at
		FROM services
		WHERE actif = TRUE
	`
	args := make([]any, 0, 3)

	if filter.Categorie != "" {
		query += " AND categorie = ?"
		args = append(args, filter.Categorie)
	}
	if filter.Ville != "" {
		query += " AND ville = ?"
		args = append(args, filter.Ville)
	}
	if filter.Search != "" {
		query += " AND (titre LIKE ? OR description LIKE ?)"
		like := "%" + filter.Search + "%"
		args = append(args, like, like)
	}

	query += " ORDER BY created_at DESC, id DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	services := make([]Service, 0)
	for rows.Next() {
		service, err := scanService(rows)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate services: %w", err)
	}

	return services, nil
}

func (s *Store) GetService(ctx context.Context, id int) (Service, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, provider_id, titre, description, categorie, duree_minutes, credits, ville, actif, created_at
		FROM services
		WHERE id = ? AND actif = TRUE
	`, id)

	service, err := scanService(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Service{}, ErrNotFound
	}
	if err != nil {
		return Service{}, err
	}

	return service, nil
}

func (s *Store) UpdateService(ctx context.Context, id int, userID int, input UpdateServiceInput) (Service, error) {
	input = normalizeUpdateServiceInput(input)
	if err := validateServiceInput(input.Titre, input.Categorie, input.DureeMinutes, input.Credits); err != nil {
		return Service{}, err
	}

	existing, err := s.GetService(ctx, id)
	if err != nil {
		return Service{}, err
	}
	if existing.ProviderID != userID {
		return Service{}, ErrForbidden
	}

	actif := existing.Actif
	if input.Actif != nil {
		actif = *input.Actif
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE services
		SET titre = ?, description = ?, categorie = ?, duree_minutes = ?, credits = ?, ville = ?, actif = ?
		WHERE id = ?
	`, input.Titre, nullableString(input.Description), input.Categorie, input.DureeMinutes, input.Credits, nullableString(input.Ville), actif, id)
	if err != nil {
		return Service{}, fmt.Errorf("update service: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return Service{}, fmt.Errorf("read affected rows: %w", err)
	}
	if rows == 0 {
		return Service{}, ErrNotFound
	}
	if !actif {
		return Service{}, ErrNotFound
	}

	return s.GetService(ctx, id)
}

func (s *Store) DeleteService(ctx context.Context, id int, userID int) error {
	existing, err := s.GetService(ctx, id)
	if err != nil {
		return err
	}
	if existing.ProviderID != userID {
		return ErrForbidden
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE services
		SET actif = FALSE
		WHERE id = ?
	`, id)
	if err != nil {
		return fmt.Errorf("delete service: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) validateServiceProvider(ctx context.Context, providerID int) error {
	if _, err := s.getUserWithoutSkills(ctx, providerID); err != nil {
		return err
	}

	var skillsCount int
	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM user_skills
		WHERE user_id = ?
	`, providerID).Scan(&skillsCount); err != nil {
		return fmt.Errorf("count provider skills: %w", err)
	}
	if skillsCount == 0 {
		return fmt.Errorf("%w: provider must define at least one skill before publishing a service", ErrInvalidInput)
	}

	return nil
}

func normalizeCreateServiceInput(input CreateServiceInput) CreateServiceInput {
	input.Titre = strings.TrimSpace(input.Titre)
	input.Description = strings.TrimSpace(input.Description)
	input.Categorie = strings.TrimSpace(input.Categorie)
	input.Ville = strings.TrimSpace(input.Ville)
	return input
}

func normalizeUpdateServiceInput(input UpdateServiceInput) UpdateServiceInput {
	input.Titre = strings.TrimSpace(input.Titre)
	input.Description = strings.TrimSpace(input.Description)
	input.Categorie = strings.TrimSpace(input.Categorie)
	input.Ville = strings.TrimSpace(input.Ville)
	return input
}

func validateServiceInput(titre string, categorie string, dureeMinutes int, credits int) error {
	if titre == "" {
		return fmt.Errorf("%w: titre is required", ErrInvalidInput)
	}
	if !validServiceCategory(categorie) {
		return fmt.Errorf("%w: invalid service categorie", ErrInvalidInput)
	}
	if dureeMinutes <= 0 {
		return fmt.Errorf("%w: duree_minutes must be greater than 0", ErrInvalidInput)
	}
	if credits <= 0 {
		return fmt.Errorf("%w: credits must be greater than 0", ErrInvalidInput)
	}
	return nil
}

func validServiceCategory(category string) bool {
	switch category {
	case "Informatique", "Jardinage", "Bricolage", "Cuisine", "Musique",
		"Langues", "Sport", "Tutorat", "Déménagement", "Photographie",
		"Animalier", "Couture", "Autre":
		return true
	default:
		return false
	}
}

type serviceScanner interface {
	Scan(dest ...any) error
}

func scanService(scanner serviceScanner) (Service, error) {
	var service Service
	var description sql.NullString
	var ville sql.NullString
	var createdAt time.Time

	err := scanner.Scan(
		&service.ID,
		&service.ProviderID,
		&service.Titre,
		&description,
		&service.Categorie,
		&service.DureeMinutes,
		&service.Credits,
		&ville,
		&service.Actif,
		&createdAt,
	)
	if err != nil {
		return Service{}, err
	}

	service.Description = description.String
	service.Ville = ville.String
	service.CreatedAt = createdAt.Format(time.RFC3339)

	return service, nil
}

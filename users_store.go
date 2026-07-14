package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type CreateUserInput struct {
	Pseudo string `json:"pseudo"`
	Bio    string `json:"bio,omitempty"`
	Ville  string `json:"ville,omitempty"`
}

type UpdateUserInput struct {
	Pseudo string `json:"pseudo"`
	Bio    string `json:"bio,omitempty"`
	Ville  string `json:"ville,omitempty"`
}

func (s *Store) CreateUser(ctx context.Context, input CreateUserInput) (User, error) {
	input.Pseudo = strings.TrimSpace(input.Pseudo)
	input.Bio = strings.TrimSpace(input.Bio)
	input.Ville = strings.TrimSpace(input.Ville)
	if input.Pseudo == "" {
		return User{}, fmt.Errorf("%w: pseudo is required", ErrInvalidInput)
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO users (pseudo, bio, ville, credit_balance)
		VALUES (?, ?, ?, 10)
	`, input.Pseudo, nullableString(input.Bio), nullableString(input.Ville))
	if err != nil {
		if isDuplicateEntry(err) {
			return User{}, fmt.Errorf("%w: pseudo already exists", ErrDuplicateValue)
		}
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("read inserted user id: %w", err)
	}

	return s.GetUser(ctx, int(id))
}

func (s *Store) GetUser(ctx context.Context, id int) (User, error) {
	user, err := s.getUserWithoutSkills(ctx, id)
	if err != nil {
		return User{}, err
	}

	skills, err := s.GetUserSkills(ctx, id)
	if err != nil {
		return User{}, err
	}
	user.Skills = skills

	return user, nil
}

func (s *Store) UpdateUser(ctx context.Context, id int, input UpdateUserInput) (User, error) {
	input.Pseudo = strings.TrimSpace(input.Pseudo)
	input.Bio = strings.TrimSpace(input.Bio)
	input.Ville = strings.TrimSpace(input.Ville)
	if input.Pseudo == "" {
		return User{}, fmt.Errorf("%w: pseudo is required", ErrInvalidInput)
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET pseudo = ?, bio = ?, ville = ?
		WHERE id = ?
	`, input.Pseudo, nullableString(input.Bio), nullableString(input.Ville), id)
	if err != nil {
		if isDuplicateEntry(err) {
			return User{}, fmt.Errorf("%w: pseudo already exists", ErrDuplicateValue)
		}
		return User{}, fmt.Errorf("update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("read affected rows: %w", err)
	}
	if rows == 0 {
		return User{}, ErrNotFound
	}

	return s.GetUser(ctx, id)
}

func (s *Store) GetUserSkills(ctx context.Context, userID int) ([]Skill, error) {
	if _, err := s.getUserWithoutSkills(ctx, userID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT nom, niveau
		FROM user_skills
		WHERE user_id = ?
		ORDER BY nom
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user skills: %w", err)
	}
	defer rows.Close()

	skills := make([]Skill, 0)
	for rows.Next() {
		var skill Skill
		if err := rows.Scan(&skill.Nom, &skill.Niveau); err != nil {
			return nil, fmt.Errorf("scan user skill: %w", err)
		}
		skills = append(skills, skill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user skills: %w", err)
	}

	return skills, nil
}

func (s *Store) ReplaceUserSkills(ctx context.Context, userID int, skills []Skill) ([]Skill, error) {
	if _, err := s.getUserWithoutSkills(ctx, userID); err != nil {
		return nil, err
	}
	if err := validateSkills(skills); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin replace skills: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM user_skills WHERE user_id = ?`, userID); err != nil {
		return nil, fmt.Errorf("delete user skills: %w", err)
	}

	for _, skill := range skills {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_skills (user_id, nom, niveau)
			VALUES (?, ?, ?)
		`, userID, strings.TrimSpace(skill.Nom), strings.TrimSpace(skill.Niveau)); err != nil {
			return nil, fmt.Errorf("insert user skill: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit replace skills: %w", err)
	}

	return s.GetUserSkills(ctx, userID)
}

func (s *Store) getUserWithoutSkills(ctx context.Context, id int) (User, error) {
	var user User
	var bio sql.NullString
	var ville sql.NullString
	var createdAt time.Time

	err := s.db.QueryRowContext(ctx, `
		SELECT id, pseudo, bio, ville, credit_balance, created_at
		FROM users
		WHERE id = ?
	`, id).Scan(&user.ID, &user.Pseudo, &bio, &ville, &user.CreditBalance, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("get user: %w", err)
	}

	user.Bio = bio.String
	user.Ville = ville.String
	user.CreatedAt = createdAt.Format(time.RFC3339)

	return user, nil
}

func validateSkills(skills []Skill) error {
	seen := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		nom := strings.TrimSpace(skill.Nom)
		niveau := strings.TrimSpace(skill.Niveau)
		if nom == "" {
			return fmt.Errorf("%w: skill nom is required", ErrInvalidInput)
		}
		if !validSkillLevel(niveau) {
			return fmt.Errorf("%w: invalid skill niveau", ErrInvalidInput)
		}

		key := strings.ToLower(nom)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("%w: duplicate skill", ErrInvalidInput)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func validSkillLevel(level string) bool {
	switch level {
	case "débutant", "intermédiaire", "expert":
		return true
	default:
		return false
	}
}

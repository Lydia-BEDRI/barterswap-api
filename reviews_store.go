package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type CreateReviewInput struct {
	Note        int    `json:"note"`
	Commentaire string `json:"commentaire,omitempty"`
}

func (s *Store) CreateReview(
	ctx context.Context,
	exchangeID int,
	authorID int,
	input CreateReviewInput,
) (Review, error) {
	input.Commentaire = strings.TrimSpace(input.Commentaire)
	if err := validateReviewInput(input); err != nil {
		return Review{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Review{}, fmt.Errorf("begin create review: %w", err)
	}
	defer tx.Rollback()

	var requesterID int
	var ownerID int
	var status string
	err = tx.QueryRowContext(ctx, `
		SELECT requester_id, owner_id, status
		FROM exchanges
		WHERE id = ?
		FOR UPDATE
	`, exchangeID).Scan(&requesterID, &ownerID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return Review{}, ErrNotFound
	}
	if err != nil {
		return Review{}, fmt.Errorf("get exchange for review: %w", err)
	}
	if status != exchangeStatusCompleted {
		return Review{}, fmt.Errorf("%w: only completed exchanges can be reviewed", ErrInvalidInput)
	}

	var targetID int
	switch authorID {
	case requesterID:
		targetID = ownerID
	case ownerID:
		targetID = requesterID
	default:
		return Review{}, ErrForbidden
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO reviews (exchange_id, author_id, target_id, note, commentaire)
		VALUES (?, ?, ?, ?, ?)
	`, exchangeID, authorID, targetID, input.Note, nullableString(input.Commentaire))
	if isDuplicateEntry(err) {
		return Review{}, fmt.Errorf("%w: review already exists for this exchange", ErrInvalidInput)
	}
	if err != nil {
		return Review{}, fmt.Errorf("insert review: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Review{}, fmt.Errorf("read inserted review id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Review{}, fmt.Errorf("commit create review: %w", err)
	}

	return s.getReview(ctx, int(id))
}

func (s *Store) ListUserReviews(ctx context.Context, userID int) ([]Review, error) {
	if _, err := s.getUserWithoutSkills(ctx, userID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, exchange_id, author_id, target_id, note, commentaire, created_at
		FROM reviews
		WHERE target_id = ?
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list user reviews: %w", err)
	}
	defer rows.Close()

	return scanReviews(rows)
}

func (s *Store) ListServiceReviews(ctx context.Context, serviceID int) ([]Review, error) {
	var id int
	err := s.db.QueryRowContext(ctx, `
		SELECT id
		FROM services
		WHERE id = ?
	`, serviceID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get service for reviews: %w", err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT r.id, r.exchange_id, r.author_id, r.target_id, r.note, r.commentaire, r.created_at
		FROM reviews r
		JOIN exchanges e ON e.id = r.exchange_id
		WHERE e.service_id = ? AND r.target_id = e.owner_id
		ORDER BY r.created_at DESC, r.id DESC
	`, serviceID)
	if err != nil {
		return nil, fmt.Errorf("list service reviews: %w", err)
	}
	defer rows.Close()

	return scanReviews(rows)
}

func (s *Store) getReview(ctx context.Context, id int) (Review, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, exchange_id, author_id, target_id, note, commentaire, created_at
		FROM reviews
		WHERE id = ?
	`, id)

	review, err := scanReview(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Review{}, ErrNotFound
	}
	if err != nil {
		return Review{}, err
	}
	return review, nil
}

func validateReviewInput(input CreateReviewInput) error {
	if input.Note < 1 || input.Note > 5 {
		return fmt.Errorf("%w: note must be between 1 and 5", ErrInvalidInput)
	}
	return nil
}

type reviewScanner interface {
	Scan(dest ...any) error
}

func scanReview(scanner reviewScanner) (Review, error) {
	var review Review
	var commentaire sql.NullString
	var createdAt time.Time

	if err := scanner.Scan(
		&review.ID,
		&review.ExchangeID,
		&review.AuthorID,
		&review.TargetID,
		&review.Note,
		&commentaire,
		&createdAt,
	); err != nil {
		return Review{}, err
	}
	review.Commentaire = commentaire.String
	review.CreatedAt = createdAt.Format(time.RFC3339)
	return review, nil
}

func scanReviews(rows *sql.Rows) ([]Review, error) {
	reviews := make([]Review, 0)
	for rows.Next() {
		review, err := scanReview(rows)
		if err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, review)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate reviews: %w", err)
	}
	return reviews, nil
}

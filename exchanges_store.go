package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	exchangeStatusPending   = "pending"
	exchangeStatusAccepted  = "accepted"
	exchangeStatusRejected  = "rejected"
	exchangeStatusCancelled = "cancelled"
	exchangeStatusCompleted = "completed"
)

type CreateExchangeInput struct {
	ServiceID int `json:"service_id"`
}

type ExchangeFilter struct {
	Status string
}

type exchangeRow struct {
	Exchange
	Credits int
}

func (s *Store) CreateExchange(ctx context.Context, requesterID int, input CreateExchangeInput) (Exchange, error) {
	if input.ServiceID <= 0 {
		return Exchange{}, fmt.Errorf("%w: service_id is required", ErrInvalidInput)
	}
	if _, err := s.getUserWithoutSkills(ctx, requesterID); err != nil {
		return Exchange{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Exchange{}, fmt.Errorf("begin create exchange: %w", err)
	}
	defer tx.Rollback()

	var ownerID int
	var credits int
	err = tx.QueryRowContext(ctx, `
		SELECT provider_id, credits
		FROM services
		WHERE id = ? AND actif = TRUE
		FOR UPDATE
	`, input.ServiceID).Scan(&ownerID, &credits)
	if errors.Is(err, sql.ErrNoRows) {
		return Exchange{}, ErrNotFound
	}
	if err != nil {
		return Exchange{}, fmt.Errorf("get service for exchange: %w", err)
	}
	if ownerID == requesterID {
		return Exchange{}, fmt.Errorf("%w: cannot request your own service", ErrInvalidInput)
	}

	if err := ensureNoActiveExchange(ctx, tx, input.ServiceID, 0); err != nil {
		return Exchange{}, err
	}

	balance, err := creditBalance(ctx, tx, requesterID)
	if err != nil {
		return Exchange{}, err
	}
	if balance < credits {
		return Exchange{}, fmt.Errorf("%w: insufficient credits", ErrInvalidInput)
	}

	result, err := tx.ExecContext(ctx, `
		INSERT INTO exchanges (service_id, requester_id, owner_id, status)
		VALUES (?, ?, ?, ?)
	`, input.ServiceID, requesterID, ownerID, exchangeStatusPending)
	if err != nil {
		return Exchange{}, fmt.Errorf("insert exchange: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Exchange{}, fmt.Errorf("read inserted exchange id: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Exchange{}, fmt.Errorf("commit create exchange: %w", err)
	}

	return s.GetExchange(ctx, int(id))
}

func (s *Store) ListExchanges(ctx context.Context, userID int, filter ExchangeFilter) ([]Exchange, error) {
	if _, err := s.getUserWithoutSkills(ctx, userID); err != nil {
		return nil, err
	}

	filter.Status = strings.TrimSpace(filter.Status)
	if filter.Status != "" && !validExchangeStatus(filter.Status) {
		return nil, fmt.Errorf("%w: invalid exchange status", ErrInvalidInput)
	}

	query := `
		SELECT id, service_id, requester_id, owner_id, status, created_at, updated_at
		FROM exchanges
		WHERE (requester_id = ? OR owner_id = ?)
	`
	args := []any{userID, userID}

	if filter.Status != "" {
		query += " AND status = ?"
		args = append(args, filter.Status)
	}

	query += " ORDER BY updated_at DESC, id DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list exchanges: %w", err)
	}
	defer rows.Close()

	exchanges := make([]Exchange, 0)
	for rows.Next() {
		exchange, err := scanExchange(rows)
		if err != nil {
			return nil, err
		}
		exchanges = append(exchanges, exchange)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exchanges: %w", err)
	}

	return exchanges, nil
}

func (s *Store) GetExchange(ctx context.Context, id int) (Exchange, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, service_id, requester_id, owner_id, status, created_at, updated_at
		FROM exchanges
		WHERE id = ?
	`, id)

	exchange, err := scanExchange(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Exchange{}, ErrNotFound
	}
	if err != nil {
		return Exchange{}, err
	}

	return exchange, nil
}

func (s *Store) GetExchangeForUser(ctx context.Context, id int, userID int) (Exchange, error) {
	exchange, err := s.GetExchange(ctx, id)
	if err != nil {
		return Exchange{}, err
	}
	if !isExchangeParticipant(exchange, userID) {
		return Exchange{}, ErrForbidden
	}
	return exchange, nil
}

func (s *Store) AcceptExchange(ctx context.Context, exchangeID int, userID int) (Exchange, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Exchange{}, fmt.Errorf("begin accept exchange: %w", err)
	}
	defer tx.Rollback()

	row, err := getExchangeRowForUpdate(ctx, tx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if row.OwnerID != userID {
		return Exchange{}, ErrForbidden
	}
	if row.Status != exchangeStatusPending {
		return Exchange{}, fmt.Errorf("%w: only pending exchanges can be accepted", ErrInvalidInput)
	}
	if err := ensureNoActiveExchange(ctx, tx, row.ServiceID, row.ID); err != nil {
		return Exchange{}, err
	}

	balance, err := creditBalance(ctx, tx, row.RequesterID)
	if err != nil {
		return Exchange{}, err
	}
	if balance < row.Credits {
		return Exchange{}, fmt.Errorf("%w: requester has insufficient credits", ErrInvalidInput)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
		VALUES (?, ?, ?, 'spend')
	`, row.RequesterID, row.ID, -row.Credits); err != nil {
		return Exchange{}, fmt.Errorf("block requester credits: %w", err)
	}

	if err := updateExchangeStatus(ctx, tx, row.ID, exchangeStatusAccepted); err != nil {
		return Exchange{}, err
	}
	if err := tx.Commit(); err != nil {
		return Exchange{}, fmt.Errorf("commit accept exchange: %w", err)
	}

	return s.GetExchange(ctx, exchangeID)
}

func (s *Store) RejectExchange(ctx context.Context, exchangeID int, userID int) (Exchange, error) {
	return s.transitionExchange(ctx, exchangeID, userID, exchangeStatusPending, exchangeStatusRejected, ownerOnly)
}

func (s *Store) CompleteExchange(ctx context.Context, exchangeID int, userID int) (Exchange, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Exchange{}, fmt.Errorf("begin complete exchange: %w", err)
	}
	defer tx.Rollback()

	row, err := getExchangeRowForUpdate(ctx, tx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if !isExchangeParticipant(row.Exchange, userID) {
		return Exchange{}, ErrForbidden
	}
	if row.Status != exchangeStatusAccepted {
		return Exchange{}, fmt.Errorf("%w: only accepted exchanges can be completed", ErrInvalidInput)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
		VALUES (?, ?, ?, 'earn')
	`, row.OwnerID, row.ID, row.Credits); err != nil {
		return Exchange{}, fmt.Errorf("transfer credits to owner: %w", err)
	}

	if err := updateExchangeStatus(ctx, tx, row.ID, exchangeStatusCompleted); err != nil {
		return Exchange{}, err
	}
	if err := tx.Commit(); err != nil {
		return Exchange{}, fmt.Errorf("commit complete exchange: %w", err)
	}

	return s.GetExchange(ctx, exchangeID)
}

func (s *Store) CancelExchange(ctx context.Context, exchangeID int, userID int) (Exchange, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Exchange{}, fmt.Errorf("begin cancel exchange: %w", err)
	}
	defer tx.Rollback()

	row, err := getExchangeRowForUpdate(ctx, tx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if !isExchangeParticipant(row.Exchange, userID) {
		return Exchange{}, ErrForbidden
	}
	if row.Status != exchangeStatusPending && row.Status != exchangeStatusAccepted {
		return Exchange{}, fmt.Errorf("%w: only pending or accepted exchanges can be cancelled", ErrInvalidInput)
	}

	if row.Status == exchangeStatusAccepted {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
			VALUES (?, ?, ?, 'refund')
		`, row.RequesterID, row.ID, row.Credits); err != nil {
			return Exchange{}, fmt.Errorf("refund requester credits: %w", err)
		}
	}

	if err := updateExchangeStatus(ctx, tx, row.ID, exchangeStatusCancelled); err != nil {
		return Exchange{}, err
	}
	if err := tx.Commit(); err != nil {
		return Exchange{}, fmt.Errorf("commit cancel exchange: %w", err)
	}

	return s.GetExchange(ctx, exchangeID)
}

type participantRule int

const (
	ownerOnly participantRule = iota
)

func (s *Store) transitionExchange(
	ctx context.Context,
	exchangeID int,
	userID int,
	fromStatus string,
	toStatus string,
	rule participantRule,
) (Exchange, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Exchange{}, fmt.Errorf("begin exchange transition: %w", err)
	}
	defer tx.Rollback()

	row, err := getExchangeRowForUpdate(ctx, tx, exchangeID)
	if err != nil {
		return Exchange{}, err
	}
	if rule == ownerOnly && row.OwnerID != userID {
		return Exchange{}, ErrForbidden
	}
	if row.Status != fromStatus {
		return Exchange{}, fmt.Errorf("%w: invalid exchange status transition", ErrInvalidInput)
	}
	if err := updateExchangeStatus(ctx, tx, row.ID, toStatus); err != nil {
		return Exchange{}, err
	}
	if err := tx.Commit(); err != nil {
		return Exchange{}, fmt.Errorf("commit exchange transition: %w", err)
	}

	return s.GetExchange(ctx, exchangeID)
}

func getExchangeRowForUpdate(ctx context.Context, tx *sql.Tx, id int) (exchangeRow, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT e.id, e.service_id, e.requester_id, e.owner_id, e.status, e.created_at, e.updated_at, s.credits
		FROM exchanges e
		JOIN services s ON s.id = e.service_id
		WHERE e.id = ?
		FOR UPDATE
	`, id)

	exchange, err := scanExchangeWithCredits(row)
	if errors.Is(err, sql.ErrNoRows) {
		return exchangeRow{}, ErrNotFound
	}
	if err != nil {
		return exchangeRow{}, err
	}

	return exchange, nil
}

func ensureNoActiveExchange(ctx context.Context, tx *sql.Tx, serviceID int, exceptExchangeID int) error {
	var activeID int
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM exchanges
		WHERE service_id = ?
			AND status IN ('pending', 'accepted')
			AND id <> ?
		LIMIT 1
		FOR UPDATE
	`, serviceID, exceptExchangeID).Scan(&activeID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("check active exchange: %w", err)
	}
	return fmt.Errorf("%w: service already has an active exchange", ErrConflict)
}

func creditBalance(ctx context.Context, tx *sql.Tx, userID int) (int, error) {
	var balance int
	if err := tx.QueryRowContext(ctx, `
		SELECT u.credit_balance + COALESCE(SUM(
			CASE ct.type
				WHEN 'spend' THEN -ABS(ct.montant)
				WHEN 'earn' THEN ABS(ct.montant)
				WHEN 'refund' THEN ABS(ct.montant)
				ELSE ct.montant
			END
		), 0)
		FROM users u
		LEFT JOIN credit_transactions ct ON ct.user_id = u.id
		WHERE u.id = ?
		GROUP BY u.id, u.credit_balance
	`, userID).Scan(&balance); err != nil {
		return 0, fmt.Errorf("read credit balance: %w", err)
	}
	return balance, nil
}

func updateExchangeStatus(ctx context.Context, tx *sql.Tx, id int, status string) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE exchanges
		SET status = ?
		WHERE id = ?
	`, status, id)
	if err != nil {
		return fmt.Errorf("update exchange status: %w", err)
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

func validExchangeStatus(status string) bool {
	switch status {
	case exchangeStatusPending, exchangeStatusAccepted, exchangeStatusRejected,
		exchangeStatusCancelled, exchangeStatusCompleted:
		return true
	default:
		return false
	}
}

func isExchangeParticipant(exchange Exchange, userID int) bool {
	return exchange.RequesterID == userID || exchange.OwnerID == userID
}

type exchangeScanner interface {
	Scan(dest ...any) error
}

func scanExchange(scanner exchangeScanner) (Exchange, error) {
	var exchange Exchange
	var createdAt time.Time
	var updatedAt time.Time

	err := scanner.Scan(
		&exchange.ID,
		&exchange.ServiceID,
		&exchange.RequesterID,
		&exchange.OwnerID,
		&exchange.Status,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return Exchange{}, err
	}

	exchange.CreatedAt = createdAt.Format(time.RFC3339)
	exchange.UpdatedAt = updatedAt.Format(time.RFC3339)

	return exchange, nil
}

func scanExchangeWithCredits(scanner exchangeScanner) (exchangeRow, error) {
	var row exchangeRow
	var createdAt time.Time
	var updatedAt time.Time

	err := scanner.Scan(
		&row.ID,
		&row.ServiceID,
		&row.RequesterID,
		&row.OwnerID,
		&row.Status,
		&createdAt,
		&updatedAt,
		&row.Credits,
	)
	if err != nil {
		return exchangeRow{}, err
	}

	row.CreatedAt = createdAt.Format(time.RFC3339)
	row.UpdatedAt = updatedAt.Format(time.RFC3339)

	return row, nil
}

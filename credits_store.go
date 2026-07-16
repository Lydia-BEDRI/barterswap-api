package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

const (
	creditTransactionEarn   = "earn"
	creditTransactionSpend  = "spend"
	creditTransactionRefund = "refund"
)

type CreditTransactionFilter struct {
	Type       string
	ExchangeID int
}

func (s *Store) ListCreditTransactions(
	ctx context.Context,
	userID int,
	filter CreditTransactionFilter,
) ([]CreditTransaction, error) {
	if _, err := s.getUserWithoutSkills(ctx, userID); err != nil {
		return nil, err
	}
	if filter.Type != "" && !validCreditTransactionType(filter.Type) {
		return nil, fmt.Errorf("%w: invalid credit transaction type", ErrInvalidInput)
	}
	if filter.ExchangeID < 0 {
		return nil, fmt.Errorf("%w: invalid exchange_id", ErrInvalidInput)
	}

	query := `
		SELECT id, user_id, exchange_id, montant, type, created_at
		FROM credit_transactions
		WHERE user_id = ?
	`
	args := []any{userID}
	if filter.Type != "" {
		query += " AND type = ?"
		args = append(args, filter.Type)
	}
	if filter.ExchangeID > 0 {
		query += " AND exchange_id = ?"
		args = append(args, filter.ExchangeID)
	}
	query += " ORDER BY created_at DESC, id DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list credit transactions: %w", err)
	}
	defer rows.Close()

	transactions := make([]CreditTransaction, 0)
	for rows.Next() {
		transaction, err := scanCreditTransaction(rows)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate credit transactions: %w", err)
	}

	return transactions, nil
}

func lockCreditBalance(ctx context.Context, tx *sql.Tx, userID int) (int, error) {
	var balance int
	err := tx.QueryRowContext(ctx, `
		SELECT credit_balance
		FROM users
		WHERE id = ?
		FOR UPDATE
	`, userID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock credit balance: %w", err)
	}
	return balance, nil
}

func changeCreditBalance(ctx context.Context, tx *sql.Tx, userID int, delta int) (int, error) {
	balance, err := lockCreditBalance(ctx, tx, userID)
	if err != nil {
		return 0, err
	}

	next, err := nextCreditBalance(balance, delta)
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET credit_balance = ?
		WHERE id = ?
	`, next, userID); err != nil {
		return 0, fmt.Errorf("update credit balance: %w", err)
	}
	return next, nil
}

func nextCreditBalance(balance int, delta int) (int, error) {
	next := balance + delta
	if next < 0 {
		return 0, ErrInsufficientCredits
	}
	return next, nil
}

func appendCreditTransaction(
	ctx context.Context,
	tx *sql.Tx,
	userID int,
	exchangeID int,
	amount int,
	transactionType string,
) error {
	if err := validateCreditTransaction(amount, transactionType); err != nil {
		return err
	}

	_, err := tx.ExecContext(ctx, `
		INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
		VALUES (?, ?, ?, ?)
	`, userID, exchangeID, amount, transactionType)
	if isDuplicateEntry(err) {
		return fmt.Errorf("%w: credit operation already journaled", ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("append credit transaction: %w", err)
	}
	return nil
}

func getExchangeCreditTransactionAmount(
	ctx context.Context,
	tx *sql.Tx,
	userID int,
	exchangeID int,
	transactionType string,
) (int, error) {
	var amount int
	err := tx.QueryRowContext(ctx, `
		SELECT montant
		FROM credit_transactions
		WHERE user_id = ? AND exchange_id = ? AND type = ?
		FOR UPDATE
	`, userID, exchangeID, transactionType).Scan(&amount)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: missing credit block", ErrConflict)
	}
	if err != nil {
		return 0, fmt.Errorf("get exchange credit transaction: %w", err)
	}
	return amount, nil
}

func validateCreditTransaction(amount int, transactionType string) error {
	switch transactionType {
	case creditTransactionSpend:
		if amount >= 0 {
			return fmt.Errorf("%w: spend amount must be negative", ErrInvalidInput)
		}
	case creditTransactionEarn, creditTransactionRefund:
		if amount <= 0 {
			return fmt.Errorf("%w: %s amount must be positive", ErrInvalidInput, transactionType)
		}
	default:
		return fmt.Errorf("%w: invalid credit transaction type", ErrInvalidInput)
	}
	return nil
}

func validCreditTransactionType(transactionType string) bool {
	switch transactionType {
	case creditTransactionEarn, creditTransactionSpend, creditTransactionRefund:
		return true
	default:
		return false
	}
}

type creditTransactionScanner interface {
	Scan(dest ...any) error
}

func scanCreditTransaction(scanner creditTransactionScanner) (CreditTransaction, error) {
	var transaction CreditTransaction
	var createdAt time.Time

	if err := scanner.Scan(
		&transaction.ID,
		&transaction.UserID,
		&transaction.ExchangeID,
		&transaction.Montant,
		&transaction.Type,
		&createdAt,
	); err != nil {
		return CreditTransaction{}, fmt.Errorf("scan credit transaction: %w", err)
	}
	transaction.CreatedAt = createdAt.Format(time.RFC3339)
	return transaction, nil
}

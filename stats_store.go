package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

func (s *Store) GetUserStats(ctx context.Context, userID int) (UserStats, error) {
	var stats UserStats
	err := s.db.QueryRowContext(ctx, `
		SELECT
			u.id,
			(
				SELECT COUNT(*)
				FROM services s
				WHERE s.provider_id = u.id AND s.actif = TRUE
			),
			(
				SELECT COUNT(*)
				FROM exchanges e
				WHERE (e.requester_id = u.id OR e.owner_id = u.id)
					AND e.status = 'completed'
			),
			u.credit_balance,
			(
				SELECT COALESCE(AVG(r.note), 0)
				FROM reviews r
				WHERE r.target_id = u.id
			),
			(
				SELECT COUNT(*)
				FROM reviews r
				WHERE r.target_id = u.id
			),
			(
				SELECT COALESCE(SUM(ct.montant), 0)
				FROM credit_transactions ct
				WHERE ct.user_id = u.id AND ct.type = 'earn'
			),
			(
				SELECT COALESCE(SUM(-ct.montant), 0)
				FROM credit_transactions ct
				WHERE ct.user_id = u.id AND ct.type = 'spend'
			)
		FROM users u
		WHERE u.id = ?
	`, userID).Scan(
		&stats.UserID,
		&stats.ServicesActifs,
		&stats.EchangesCompletes,
		&stats.CreditBalance,
		&stats.NoteMoyenne,
		&stats.NbAvis,
		&stats.TotalGagne,
		&stats.TotalDepense,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return UserStats{}, ErrNotFound
	}
	if err != nil {
		return UserStats{}, fmt.Errorf("get user stats: %w", err)
	}
	return stats, nil
}

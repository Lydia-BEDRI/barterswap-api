-- Upgrade a database created before the credit-system feature.
-- Run this migration exactly once before deploying the new API version.

SET NAMES utf8mb4 COLLATE utf8mb4_unicode_ci;

-- Orphan rows are not part of the subject model: every transaction belongs
-- to an exchange.
DELETE FROM credit_transactions
WHERE exchange_id IS NULL;

-- Remove journal operations that are impossible for the exchange status.
DELETE ct
FROM credit_transactions ct
JOIN exchanges e ON e.id = ct.exchange_id
WHERE (ct.type = 'earn' AND e.status <> 'completed')
   OR (ct.type = 'spend' AND e.status NOT IN ('accepted', 'completed', 'cancelled'))
   OR (ct.type = 'refund' AND e.status <> 'cancelled');

-- A spend/refund always belongs to the requester; an earn always belongs to
-- the service owner.
UPDATE credit_transactions ct
JOIN exchanges e ON e.id = ct.exchange_id
SET ct.user_id = CASE
  WHEN ct.type = 'earn' THEN e.owner_id
  ELSE e.requester_id
END;

-- Repair missing legacy entries for exchanges that currently hold or already
-- transferred credits.
INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
SELECT e.requester_id, e.id, -s.credits, 'spend'
FROM exchanges e
JOIN services s ON s.id = e.service_id
WHERE e.status IN ('accepted', 'completed')
  AND NOT EXISTS (
    SELECT 1
    FROM credit_transactions ct
    WHERE ct.exchange_id = e.id AND ct.type = 'spend'
  );

INSERT INTO credit_transactions (user_id, exchange_id, montant, type)
SELECT e.owner_id, e.id, s.credits, 'earn'
FROM exchanges e
JOIN services s ON s.id = e.service_id
WHERE e.status = 'completed'
  AND NOT EXISTS (
    SELECT 1
    FROM credit_transactions ct
    WHERE ct.exchange_id = e.id AND ct.type = 'earn'
  );

UPDATE credit_transactions
SET montant = CASE
  WHEN type = 'spend' THEN -ABS(montant)
  ELSE ABS(montant)
END;

-- The legacy implementation treated users.credit_balance as an opening
-- balance. Convert it once to the new, directly spendable balance.
UPDATE users u
LEFT JOIN (
  SELECT user_id, SUM(montant) AS journal_delta
  FROM credit_transactions
  GROUP BY user_id
) journal ON journal.user_id = u.id
SET u.credit_balance = u.credit_balance + COALESCE(journal.journal_delta, 0);

ALTER TABLE credit_transactions
  DROP FOREIGN KEY credit_transactions_ibfk_2;

ALTER TABLE credit_transactions
  MODIFY COLUMN exchange_id INT NOT NULL,
  ADD UNIQUE KEY unique_exchange_credit_operation (exchange_id, user_id, type),
  ADD CONSTRAINT chk_credit_transaction_amount CHECK (
    (type = 'spend' AND montant < 0)
    OR (type IN ('earn', 'refund') AND montant > 0)
  );

ALTER TABLE credit_transactions
  ADD CONSTRAINT credit_transactions_ibfk_2
    FOREIGN KEY (exchange_id) REFERENCES exchanges(id) ON DELETE CASCADE;

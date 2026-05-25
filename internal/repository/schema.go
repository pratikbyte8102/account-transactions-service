package repository

import (
	"context"
	"database/sql"
)

const SchemaSQL = `
CREATE TABLE IF NOT EXISTS accounts (
	account_id BIGSERIAL PRIMARY KEY,
	document_number TEXT NOT NULL UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS operation_types (
	operation_type_id SMALLINT PRIMARY KEY,
	description TEXT NOT NULL
);

INSERT INTO operation_types (operation_type_id, description)
VALUES
	(1, 'Normal Purchase'),
	(2, 'Purchase with installments'),
	(3, 'Withdrawal'),
	(4, 'Credit Voucher')
ON CONFLICT (operation_type_id) DO UPDATE
SET description = EXCLUDED.description;

CREATE TABLE IF NOT EXISTS transactions (
	transaction_id BIGSERIAL PRIMARY KEY,
	account_id BIGINT NOT NULL REFERENCES accounts(account_id),
	operation_type_id SMALLINT NOT NULL REFERENCES operation_types(operation_type_id),
	amount NUMERIC(12, 2) NOT NULL CHECK (amount <> 0),
	event_date TIMESTAMPTZ NOT NULL
);
`

func Migrate(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, SchemaSQL)
	return err
}

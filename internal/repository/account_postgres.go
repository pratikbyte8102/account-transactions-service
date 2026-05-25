package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"

	"account-transactions-service/internal/domain"
	"account-transactions-service/internal/service"
)

type AccountPostgresRepository struct {
	db *sql.DB
}

func NewAccountPostgresRepository(db *sql.DB) *AccountPostgresRepository {
	return &AccountPostgresRepository{db: db}
}

func (r *AccountPostgresRepository) Create(ctx context.Context, documentNumber string) (domain.Account, error) {
	const query = `
		INSERT INTO accounts (document_number)
		VALUES ($1)
		RETURNING account_id, document_number`

	var account domain.Account
	err := r.db.QueryRowContext(ctx, query, documentNumber).Scan(&account.ID, &account.DocumentNumber)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.Account{}, service.ErrConflict
		}
		return domain.Account{}, err
	}
	return account, nil
}

func (r *AccountPostgresRepository) FindByID(ctx context.Context, id int64) (domain.Account, error) {
	const query = `
		SELECT account_id, document_number
		FROM accounts
		WHERE account_id = $1`

	var account domain.Account
	err := r.db.QueryRowContext(ctx, query, id).Scan(&account.ID, &account.DocumentNumber)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Account{}, service.ErrNotFound
	}
	if err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

func (r *AccountPostgresRepository) Exists(ctx context.Context, id int64) (bool, error) {
	const query = `SELECT EXISTS (SELECT 1 FROM accounts WHERE account_id = $1)`

	var exists bool
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

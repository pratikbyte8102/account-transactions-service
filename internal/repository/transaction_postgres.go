package repository

import (
	"context"
	"database/sql"

	"github.com/lib/pq"

	"account-transactions-service/internal/domain"
	"account-transactions-service/internal/service"
)

type TransactionPostgresRepository struct {
	db *sql.DB
}

func NewTransactionPostgresRepository(db *sql.DB) *TransactionPostgresRepository {
	return &TransactionPostgresRepository{db: db}
}

func (r *TransactionPostgresRepository) Create(ctx context.Context, tx domain.NewTransaction) (domain.Transaction, error) {
	const query = `
		INSERT INTO transactions (account_id, operation_type_id, amount, event_date)
		VALUES ($1, $2, $3, $4)
		RETURNING transaction_id, account_id, operation_type_id, amount, event_date`

	var created domain.Transaction
	var operationTypeID int64
	err := r.db.QueryRowContext(ctx, query, tx.AccountID, tx.OperationTypeID, tx.Amount, tx.EventDate).
		Scan(&created.ID, &created.AccountID, &operationTypeID, &created.Amount, &created.EventDate)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23503" {
			return domain.Transaction{}, service.ErrNotFound
		}
		return domain.Transaction{}, err
	}
	created.OperationTypeID = domain.OperationTypeID(operationTypeID)
	return created, nil
}

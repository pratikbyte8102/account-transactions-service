package service

import (
	"context"
	"errors"
	"time"

	"account-transactions-service/internal/domain"
)

type AccountRepository interface {
	Create(ctx context.Context, documentNumber string) (domain.Account, error)
	FindByID(ctx context.Context, id int64) (domain.Account, error)
	Exists(ctx context.Context, id int64) (bool, error)
}

type TransactionRepository interface {
	Create(ctx context.Context, tx domain.NewTransaction) (domain.Transaction, error)
}

type Service struct {
	accounts     AccountRepository
	transactions TransactionRepository
	now          func() time.Time
}

func New(accounts AccountRepository, transactions TransactionRepository) *Service {
	return &Service{
		accounts:     accounts,
		transactions: transactions,
		now:          time.Now,
	}
}

func (s *Service) SetClockForTest(now func() time.Time) {
	s.now = now
}

func (s *Service) CreateAccount(ctx context.Context, documentNumber string) (domain.Account, error) {
	documentNumber = domain.NormalizeDocumentNumber(documentNumber)
	if documentNumber == "" {
		return domain.Account{}, ValidationError{Message: "document_number is required"}
	}

	account, err := s.accounts.Create(ctx, documentNumber)
	if err != nil {
		return domain.Account{}, err
	}
	return account, nil
}

func (s *Service) GetAccount(ctx context.Context, id int64) (domain.Account, error) {
	if id <= 0 {
		return domain.Account{}, ValidationError{Message: "account_id must be positive"}
	}
	return s.accounts.FindByID(ctx, id)
}

func (s *Service) CreateTransaction(ctx context.Context, accountID int64, operationTypeID domain.OperationTypeID, amount float64) (domain.Transaction, error) {
	if accountID <= 0 {
		return domain.Transaction{}, ValidationError{Message: "account_id must be positive"}
	}
	if !domain.ValidOperationType(operationTypeID) {
		return domain.Transaction{}, ValidationError{Message: "operation_type_id is invalid"}
	}
	if operationTypeID == domain.OperationTypeCreditVoucher && amount <= 0 {
		return domain.Transaction{}, ValidationError{Message: "amount must be positive for credit voucher"}
	}
	if operationTypeID != domain.OperationTypeCreditVoucher && amount >= 0 {
		return domain.Transaction{}, ValidationError{Message: "amount must be negative for purchases and withdrawals"}
	}

	exists, err := s.accounts.Exists(ctx, accountID)
	if err != nil {
		return domain.Transaction{}, err
	}
	if !exists {
		return domain.Transaction{}, ErrNotFound
	}

	tx := domain.NewTransaction{
		AccountID:       accountID,
		OperationTypeID: operationTypeID,
		Amount:          domain.NormalizeAmount(operationTypeID, amount),
		EventDate:       s.now().UTC(),
	}

	created, err := s.transactions.Create(ctx, tx)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return domain.Transaction{}, ErrNotFound
		}
		return domain.Transaction{}, err
	}
	return created, nil
}

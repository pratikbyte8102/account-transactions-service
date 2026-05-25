package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"
	"time"

	"account-transactions-service/internal/domain"
)

func TestCreateAccount(t *testing.T) {
	accounts := &fakeAccountRepository{}
	svc := New(accounts, &fakeTransactionRepository{})

	account, err := svc.CreateAccount(context.Background(), " 12345678900 ")
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	if account.ID != 1 {
		t.Fatalf("account.ID = %d, want 1", account.ID)
	}
	if account.DocumentNumber != "12345678900" {
		t.Fatalf("account.DocumentNumber = %q, want 12345678900", account.DocumentNumber)
	}
}

func TestCreateAccountRequiresDocumentNumber(t *testing.T) {
	svc := New(&fakeAccountRepository{}, &fakeTransactionRepository{})

	_, err := svc.CreateAccount(context.Background(), " ")
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("CreateAccount() error = %v, want validation", err)
	}
}

func TestGetAccountNotFound(t *testing.T) {
	svc := New(&fakeAccountRepository{}, &fakeTransactionRepository{})

	_, err := svc.GetAccount(context.Background(), 99)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetAccount() error = %v, want not found", err)
	}
}

func TestCreateTransactionNormalizesAmountByOperationType(t *testing.T) {
	fixedTime := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name            string
		operationTypeID domain.OperationTypeID
		amount          float64
		wantAmount      float64
	}{
		{name: "normal purchase", operationTypeID: domain.OperationTypeNormalPurchase, amount: -123.45, wantAmount: -123.45},
		{name: "installment purchase", operationTypeID: domain.OperationTypeInstallmentPurchase, amount: -123.45, wantAmount: -123.45},
		{name: "withdrawal", operationTypeID: domain.OperationTypeWithdrawal, amount: -123.45, wantAmount: -123.45},
		{name: "credit voucher", operationTypeID: domain.OperationTypeCreditVoucher, amount: 123.45, wantAmount: 123.45},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transactions := &fakeTransactionRepository{}
			svc := New(&fakeAccountRepository{existingIDs: map[int64]bool{1: true}}, transactions)
			svc.SetClockForTest(func() time.Time { return fixedTime })

			tx, err := svc.CreateTransaction(context.Background(), 1, tt.operationTypeID, tt.amount)
			if err != nil {
				t.Fatalf("CreateTransaction() error = %v", err)
			}

			if math.Abs(tx.Amount-tt.wantAmount) > 0.000001 {
				t.Fatalf("tx.Amount = %v, want %v", tx.Amount, tt.wantAmount)
			}
			if !tx.EventDate.Equal(fixedTime) {
				t.Fatalf("tx.EventDate = %v, want %v", tx.EventDate, fixedTime)
			}
			if transactions.created.Amount != tt.wantAmount {
				t.Fatalf("stored amount = %v, want %v", transactions.created.Amount, tt.wantAmount)
			}
		})
	}
}

func TestCreateTransactionRejectsInvalidOperationType(t *testing.T) {
	svc := New(&fakeAccountRepository{existingIDs: map[int64]bool{1: true}}, &fakeTransactionRepository{})

	_, err := svc.CreateTransaction(context.Background(), 1, domain.OperationTypeID(99), 10)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("CreateTransaction() error = %v, want validation", err)
	}
}

func TestCreateTransactionRejectsUnknownAccount(t *testing.T) {
	svc := New(&fakeAccountRepository{}, &fakeTransactionRepository{})

	_, err := svc.CreateTransaction(context.Background(), 404, domain.OperationTypeCreditVoucher, 10)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("CreateTransaction() error = %v, want not found", err)
	}
}

func TestCreateTransactionRejectsZeroAmount(t *testing.T) {
	svc := New(&fakeAccountRepository{existingIDs: map[int64]bool{1: true}}, &fakeTransactionRepository{})

	_, err := svc.CreateTransaction(context.Background(), 1, domain.OperationTypeCreditVoucher, 0)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("CreateTransaction() error = %v, want validation", err)
	}
}

func TestCreateTransactionRejectsNegativeCreditVoucherAmount(t *testing.T) {
	svc := New(&fakeAccountRepository{existingIDs: map[int64]bool{1: true}}, &fakeTransactionRepository{})

	_, err := svc.CreateTransaction(context.Background(), 1, domain.OperationTypeCreditVoucher, -123.45)
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("CreateTransaction() error = %v, want validation", err)
	}
}

func TestCreateTransactionRejectsPositivePurchaseOrWithdrawalAmount(t *testing.T) {
	tests := []domain.OperationTypeID{
		domain.OperationTypeNormalPurchase,
		domain.OperationTypeInstallmentPurchase,
		domain.OperationTypeWithdrawal,
	}

	for _, operationTypeID := range tests {
		t.Run(fmt.Sprintf("operation type %d", operationTypeID), func(t *testing.T) {
			svc := New(&fakeAccountRepository{existingIDs: map[int64]bool{1: true}}, &fakeTransactionRepository{})

			_, err := svc.CreateTransaction(context.Background(), 1, operationTypeID, 123.45)
			if !errors.Is(err, ErrValidation) {
				t.Fatalf("CreateTransaction() error = %v, want validation", err)
			}
		})
	}
}

type fakeAccountRepository struct {
	nextID      int64
	existingIDs map[int64]bool
	accounts    map[int64]domain.Account
}

func (r *fakeAccountRepository) Create(_ context.Context, documentNumber string) (domain.Account, error) {
	if r.nextID == 0 {
		r.nextID = 1
	}
	account := domain.Account{ID: r.nextID, DocumentNumber: documentNumber}
	r.nextID++
	if r.accounts == nil {
		r.accounts = map[int64]domain.Account{}
	}
	if r.existingIDs == nil {
		r.existingIDs = map[int64]bool{}
	}
	r.accounts[account.ID] = account
	r.existingIDs[account.ID] = true
	return account, nil
}

func (r *fakeAccountRepository) FindByID(_ context.Context, id int64) (domain.Account, error) {
	if r.accounts != nil {
		if account, ok := r.accounts[id]; ok {
			return account, nil
		}
	}
	return domain.Account{}, ErrNotFound
}

func (r *fakeAccountRepository) Exists(_ context.Context, id int64) (bool, error) {
	return r.existingIDs[id], nil
}

type fakeTransactionRepository struct {
	nextID  int64
	created domain.NewTransaction
}

func (r *fakeTransactionRepository) Create(_ context.Context, tx domain.NewTransaction) (domain.Transaction, error) {
	if r.nextID == 0 {
		r.nextID = 1
	}
	r.created = tx
	created := domain.Transaction{
		ID:              r.nextID,
		AccountID:       tx.AccountID,
		OperationTypeID: tx.OperationTypeID,
		Amount:          tx.Amount,
		EventDate:       tx.EventDate,
	}
	r.nextID++
	return created, nil
}

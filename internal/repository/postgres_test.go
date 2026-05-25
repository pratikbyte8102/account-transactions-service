package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"account-transactions-service/internal/domain"
)

func TestPostgresRepositoryIntegration(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("set TEST_DATABASE_URL to run PostgreSQL repository integration tests")
	}

	ctx := context.Background()
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping database: %v", err)
	}
	if err := Migrate(ctx, db); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	accounts := NewAccountPostgresRepository(db)
	transactions := NewTransactionPostgresRepository(db)

	documentNumber := fmt.Sprintf("doc-%d", time.Now().UnixNano())
	account, err := accounts.Create(ctx, documentNumber)
	if err != nil {
		t.Fatalf("create account: %v", err)
	}

	found, err := accounts.FindByID(ctx, account.ID)
	if err != nil {
		t.Fatalf("find account: %v", err)
	}
	if found.DocumentNumber != documentNumber {
		t.Fatalf("document number = %q, want %q", found.DocumentNumber, documentNumber)
	}

	eventDate := time.Now().UTC()
	tx, err := transactions.Create(ctx, domain.NewTransaction{
		AccountID:       account.ID,
		OperationTypeID: domain.OperationTypeCreditVoucher,
		Amount:          123.45,
		EventDate:       eventDate,
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if tx.ID == 0 || tx.AccountID != account.ID || tx.Amount != 123.45 {
		t.Fatalf("unexpected transaction: %#v", tx)
	}
}

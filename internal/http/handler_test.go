package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"account-transactions-service/internal/domain"
	"account-transactions-service/internal/service"
)

func TestCreateAccountHandler(t *testing.T) {
	app := &fakeApp{
		createAccount: func(_ context.Context, documentNumber string) (domain.Account, error) {
			if documentNumber != "12345678900" {
				t.Fatalf("documentNumber = %q, want 12345678900", documentNumber)
			}
			return domain.Account{ID: 1, DocumentNumber: documentNumber}, nil
		},
	}
	server := NewHandler(app, app, nil).Routes()

	req := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString(`{"document_number":"12345678900"}`))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["account_id"] != float64(1) || body["document_number"] != "12345678900" {
		t.Fatalf("body = %#v", body)
	}
}

func TestGetAccountHandler(t *testing.T) {
	app := &fakeApp{
		getAccount: func(_ context.Context, id int64) (domain.Account, error) {
			if id != 1 {
				t.Fatalf("id = %d, want 1", id)
			}
			return domain.Account{ID: 1, DocumentNumber: "12345678900"}, nil
		},
	}
	server := NewHandler(app, app, nil).Routes()

	req := httptest.NewRequest(http.MethodGet, "/accounts/1", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestCreateTransactionHandler(t *testing.T) {
	eventDate := time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
	app := &fakeApp{
		createTransaction: func(_ context.Context, accountID int64, operationTypeID domain.OperationTypeID, amount float64) (domain.Transaction, error) {
			if accountID != 1 || operationTypeID != domain.OperationTypeCreditVoucher || amount != 123.45 {
				t.Fatalf("unexpected request: accountID=%d operationTypeID=%d amount=%v", accountID, operationTypeID, amount)
			}
			return domain.Transaction{
				ID:              1,
				AccountID:       accountID,
				OperationTypeID: operationTypeID,
				Amount:          123.45,
				EventDate:       eventDate,
			}, nil
		},
	}
	server := NewHandler(app, app, nil).Routes()

	req := httptest.NewRequest(http.MethodPost, "/transactions", bytes.NewBufferString(`{"account_id":1,"operation_type_id":4,"amount":123.45}`))
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["transaction_id"] != float64(1) || body["amount"] != 123.45 {
		t.Fatalf("body = %#v", body)
	}
}

func TestHandlerValidationAndNotFoundErrors(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		body       string
		app        *fakeApp
		wantStatus int
	}{
		{
			name:       "invalid json",
			method:     http.MethodPost,
			path:       "/accounts",
			body:       `{`,
			app:        &fakeApp{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "account not found",
			method: http.MethodGet,
			path:   "/accounts/99",
			app: &fakeApp{
				getAccount: func(context.Context, int64) (domain.Account, error) {
					return domain.Account{}, service.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "transaction validation",
			method: http.MethodPost,
			path:   "/transactions",
			body:   `{"account_id":1,"operation_type_id":99,"amount":10}`,
			app: &fakeApp{
				createTransaction: func(context.Context, int64, domain.OperationTypeID, float64) (domain.Transaction, error) {
					return domain.Transaction{}, service.ValidationError{Message: "operation_type_id is invalid"}
				},
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := NewHandler(tt.app, tt.app, nil).Routes()
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestSwaggerDocumentationRoutes(t *testing.T) {
	server := NewHandler(&fakeApp{}, &fakeApp{}, nil).Routes()

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantType    string
		wantContent string
	}{
		{
			name:        "swagger ui",
			path:        "/swagger/",
			wantStatus:  http.StatusOK,
			wantType:    "text/html; charset=utf-8",
			wantContent: `url: "/swagger/openapi.yaml"`,
		},
		{
			name:        "openapi specification",
			path:        "/swagger/openapi.yaml",
			wantStatus:  http.StatusOK,
			wantType:    "application/yaml",
			wantContent: "openapi: 3.0.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			server.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if contentType := rec.Header().Get("Content-Type"); contentType != tt.wantType {
				t.Fatalf("Content-Type = %q, want %q", contentType, tt.wantType)
			}
			if !strings.Contains(rec.Body.String(), tt.wantContent) {
				t.Fatalf("body does not contain %q", tt.wantContent)
			}
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusPermanentRedirect {
		t.Fatalf("redirect status = %d, want %d", rec.Code, http.StatusPermanentRedirect)
	}
	if location := rec.Header().Get("Location"); location != "/swagger/" {
		t.Fatalf("Location = %q, want /swagger/", location)
	}
}

type fakeApp struct {
	createAccount     func(context.Context, string) (domain.Account, error)
	getAccount        func(context.Context, int64) (domain.Account, error)
	createTransaction func(context.Context, int64, domain.OperationTypeID, float64) (domain.Transaction, error)
}

func (a *fakeApp) CreateAccount(ctx context.Context, documentNumber string) (domain.Account, error) {
	if a.createAccount == nil {
		return domain.Account{}, errors.New("CreateAccount not implemented")
	}
	return a.createAccount(ctx, documentNumber)
}

func (a *fakeApp) GetAccount(ctx context.Context, id int64) (domain.Account, error) {
	if a.getAccount == nil {
		return domain.Account{}, errors.New("GetAccount not implemented")
	}
	return a.getAccount(ctx, id)
}

func (a *fakeApp) CreateTransaction(ctx context.Context, accountID int64, operationTypeID domain.OperationTypeID, amount float64) (domain.Transaction, error) {
	if a.createTransaction == nil {
		return domain.Transaction{}, errors.New("CreateTransaction not implemented")
	}
	return a.createTransaction(ctx, accountID, operationTypeID, amount)
}

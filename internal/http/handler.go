package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"account-transactions-service/internal/domain"
	"account-transactions-service/internal/service"
)

type AccountService interface {
	CreateAccount(ctx context.Context, documentNumber string) (domain.Account, error)
	GetAccount(ctx context.Context, id int64) (domain.Account, error)
}

type TransactionService interface {
	CreateTransaction(ctx context.Context, accountID int64, operationTypeID domain.OperationTypeID, amount float64) (domain.Transaction, error)
}

type Handler struct {
	accounts     AccountService
	transactions TransactionService
	logger       *slog.Logger
}

func NewHandler(accounts AccountService, transactions TransactionService, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{accounts: accounts, transactions: transactions, logger: logger}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /accounts", h.createAccount)
	mux.HandleFunc("GET /accounts/{accountId}", h.getAccount)
	mux.HandleFunc("POST /transactions", h.createTransaction)
	return mux
}

type createAccountRequest struct {
	DocumentNumber string `json:"document_number"`
}

type accountResponse struct {
	ID             int64  `json:"account_id"`
	DocumentNumber string `json:"document_number"`
}

type createTransactionRequest struct {
	AccountID       int64   `json:"account_id"`
	OperationTypeID int64   `json:"operation_type_id"`
	Amount          float64 `json:"amount"`
}

type transactionResponse struct {
	ID              int64     `json:"transaction_id"`
	AccountID       int64     `json:"account_id"`
	OperationTypeID int64     `json:"operation_type_id"`
	Amount          float64   `json:"amount"`
	EventDate       time.Time `json:"event_date"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	var req createAccountRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	account, err := h.accounts.CreateAccount(r.Context(), req.DocumentNumber)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAccountResponse(account))
}

func (h *Handler) getAccount(w http.ResponseWriter, r *http.Request) {
	accountID, err := strconv.ParseInt(r.PathValue("accountId"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "account_id must be a number")
		return
	}

	account, err := h.accounts.GetAccount(r.Context(), accountID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, toAccountResponse(account))
}

func (h *Handler) createTransaction(w http.ResponseWriter, r *http.Request) {
	var req createTransactionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	tx, err := h.transactions.CreateTransaction(
		r.Context(),
		req.AccountID,
		domain.OperationTypeID(req.OperationTypeID),
		req.Amount,
	)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toTransactionResponse(tx))
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, service.ErrNotFound):
		writeError(w, http.StatusNotFound, "resource not found")
	case errors.Is(err, service.ErrConflict):
		writeError(w, http.StatusConflict, "resource already exists")
	default:
		h.logger.Error("request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decodeJSON(r *http.Request, v any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func toAccountResponse(account domain.Account) accountResponse {
	return accountResponse{ID: account.ID, DocumentNumber: account.DocumentNumber}
}

func toTransactionResponse(tx domain.Transaction) transactionResponse {
	return transactionResponse{
		ID:              tx.ID,
		AccountID:       tx.AccountID,
		OperationTypeID: int64(tx.OperationTypeID),
		Amount:          tx.Amount,
		EventDate:       tx.EventDate,
	}
}

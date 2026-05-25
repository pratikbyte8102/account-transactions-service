package domain

import (
	"fmt"
	"math"
	"strings"
	"time"
)

type Account struct {
	ID             int64
	DocumentNumber string
}

type OperationTypeID int64

const (
	OperationTypeNormalPurchase      OperationTypeID = 1
	OperationTypeInstallmentPurchase OperationTypeID = 2
	OperationTypeWithdrawal          OperationTypeID = 3
	OperationTypeCreditVoucher       OperationTypeID = 4
)

type Transaction struct {
	ID              int64
	AccountID       int64
	OperationTypeID OperationTypeID
	Amount          float64
	EventDate       time.Time
}

type NewTransaction struct {
	AccountID       int64
	OperationTypeID OperationTypeID
	Amount          float64
	EventDate       time.Time
}

func NormalizeDocumentNumber(documentNumber string) string {
	return strings.TrimSpace(documentNumber)
}

func ValidOperationType(id OperationTypeID) bool {
	switch id {
	case OperationTypeNormalPurchase, OperationTypeInstallmentPurchase, OperationTypeWithdrawal, OperationTypeCreditVoucher:
		return true
	default:
		return false
	}
}

func NormalizeAmount(operationTypeID OperationTypeID, amount float64) float64 {
	absolute := math.Abs(amount)
	if operationTypeID == OperationTypeCreditVoucher {
		return absolute
	}
	return -absolute
}

func OperationTypeDescription(id OperationTypeID) (string, error) {
	switch id {
	case OperationTypeNormalPurchase:
		return "Normal Purchase", nil
	case OperationTypeInstallmentPurchase:
		return "Purchase with installments", nil
	case OperationTypeWithdrawal:
		return "Withdrawal", nil
	case OperationTypeCreditVoucher:
		return "Credit Voucher", nil
	default:
		return "", fmt.Errorf("invalid operation type id %d", id)
	}
}

package main

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Commission struct {
	TransactionID     int       `json:"transaction_id"`
	TransactionAmount float64   `json:"transaction_amount"`
	Currency          string    `json:"currency"`
	TransactionType   string    `json:"transaction_type"`
	CommissionAmount  float64   `json:"commission_amount"`
	Date              time.Time `json:"date"`
	Description       string    `json:"description"`
}

func CalculateCommission(t Transaction) Commission {
	logger.WithFields(logrus.Fields{
		"transaction_id": t.ID,
		"amount":         t.Amount,
		"currency":       t.Currency,
		"type":           t.Type,
	}).Info("Calculating commission for transaction")

	var commissionAmount float64
	if t.Type == "перевод" {
		switch t.Currency {
		case "USD":
			commissionAmount = t.Amount * 0.02
		case "RUB":
			commissionAmount = t.Amount * 0.05
		default:
			commissionAmount = t.Amount * 0.01 // Default commission for other currencies
		}
	} else if t.Type == "покупка" || t.Type == "пополнение" {
		commissionAmount = 0 // No commission for these types
	}

	commission := Commission{
		TransactionID:     t.ID,
		TransactionAmount: t.Amount,
		Currency:          t.Currency,
		TransactionType:   t.Type,
		CommissionAmount:  commissionAmount,
		Date:              time.Now(),
		Description:       "Calculated commission",
	}

	logger.WithFields(logrus.Fields{
		"transaction_id":    t.ID,
		"commission_amount": commissionAmount,
		"final_currency":    t.Currency,
	}).Info("Commission calculated")

	return commission
}

func handleCommissionCalculation(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req Transaction
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	commission := CalculateCommission(req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(commission)
}

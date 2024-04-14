package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	logger.Formatter = &logrus.JSONFormatter{}
	logger.Level = logrus.InfoLevel
	logger.Out = os.Stdout
}

type Transaction struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Type        string    `json:"type"`
	Category    string    `json:"category"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
}

var db *pgxpool.Pool

func main() {
	var err error
	db, err = setupDatabase()
	if err != nil {
		logger.Fatalf("Failed to setup database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/transactions", handleTransactions)
	http.HandleFunc("/transactions/", handleTransactionByID)
	http.HandleFunc("/commissions/calculate", handleCommissionCalculation)

	if err := createTables(db); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	logger.Println("Server starting on port 8080")
	logger.Fatal(http.ListenAndServe(":8080", nil))
}

func handleTransactions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getAllTransactions(w, r, db)
	case "POST":
		createTransaction(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getAllTransactions(w http.ResponseWriter, r *http.Request, db *pgxpool.Pool) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "User ID is required", http.StatusBadRequest)
		return
	}

	if _, err := strconv.Atoi(userID); err != nil {
		http.Error(w, "User ID must be an integer", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(context.Background(), "SELECT id, user_id, amount, currency, type, category, date, description FROM transactionsdb.public.transactions WHERE user_id=$1", userID)
	if err != nil {
		logger.Errorf("Error querying database: %v", err)
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var transactions []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Amount, &t.Currency, &t.Type, &t.Category, &t.Date, &t.Description); err != nil {
			http.Error(w, "Failed to read transaction data", http.StatusInternalServerError)
			return
		}
		transactions = append(transactions, t)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(transactions); err != nil {
		logger.Errorf("Error encoding response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func createTransaction(w http.ResponseWriter, r *http.Request) {
	var t Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var userID int
	err := db.QueryRow(context.Background(), "SELECT id FROM transactionsdb.public.users WHERE id = $1", t.UserID).Scan(&userID)
	if err != nil {
		username := "Default User"                           // Примерное имя, можно передавать в запросе или генерировать
		email := fmt.Sprintf("user%d@example.com", t.UserID) // Примерная почта, можно улучшить логику
		password := "example_password"                       // Пример пароля, в реальных условиях нужно хеширование

		_, err := db.Exec(context.Background(), "INSERT INTO transactionsdb.public.users (name, email, password) VALUES ($1, $2, $3)", username, email, password)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	sql := `INSERT INTO transactionsdb.public.transactions (user_id, amount, currency, type, category, date, description)
            VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`
	id := 0
	err = db.QueryRow(context.Background(), sql, t.UserID, t.Amount, t.Currency, t.Type, t.Category, t.Date, t.Description).Scan(&id)
	if err != nil {
		http.Error(w, "Failed to create transaction", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

func generateTransactionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func handleTransactionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/transactions/")
	if id == "" {
		http.Error(w, "Invalid transaction ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		getTransactionByID(w, r, id)
	case "PUT":
		updateTransaction(w, r, id)
	case "DELETE":
		deleteTransaction(w, r, id)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getTransactionByID(w http.ResponseWriter, r *http.Request, id string) {
	targetCurrency := r.URL.Query().Get("currency")

	var t Transaction
	query := "SELECT id, user_id, amount, currency, type, category, date, description FROM transactionsdb.public.transactions WHERE id=$1"
	err := db.QueryRow(context.Background(), query, id).Scan(&t.ID, &t.UserID, &t.Amount, &t.Currency, &t.Type, &t.Category, &t.Date, &t.Description)
	if err != nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}

	responseData := map[string]interface{}{
		"id":          t.ID,
		"amount":      t.Amount,
		"currency":    t.Currency,
		"type":        t.Type,
		"category":    t.Category,
		"date":        t.Date,
		"description": t.Description,
	}

	// Perform currency conversion if a target currency is specified and it is different from the transaction currency
	if targetCurrency != "" && targetCurrency != t.Currency {
		// Retrieve the API key from the environment variable
		apiKey := os.Getenv("FREECURRENCYAPI_KEY")
		if apiKey == "" {
			http.Error(w, "API key not set in environment variables", http.StatusInternalServerError)
			return
		}

		// Make a request to the currency conversion API
		conversionRate, err := getConversionRate(t.Currency, targetCurrency, apiKey)
		if err != nil {
			http.Error(w, "Failed to convert currency", http.StatusInternalServerError)
			return
		}

		// Convert the amount
		convertedAmount := t.Amount * conversionRate
		responseData["convertedAmount"] = convertedAmount
		responseData["convertedCurrency"] = targetCurrency
		responseData["rate"] = conversionRate
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseData)
}

func updateTransaction(w http.ResponseWriter, r *http.Request, id string) {
	var t Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sql := `UPDATE transactionsdb.public.transactions SET amount=$1, currency=$2, type=$3, category=$4, date=$5, description=$6 WHERE id=$7`
	_, err := db.Exec(context.Background(), sql, t.Amount, t.Currency, t.Type, t.Category, t.Date, t.Description, id)
	if err != nil {
		http.Error(w, "Failed to update transaction", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteTransaction(w http.ResponseWriter, r *http.Request, id string) {
	_, err := db.Exec(context.Background(), "DELETE FROM transactionsdb.public.transactions WHERE id=$1", id)
	if err != nil {
		http.Error(w, "Failed to delete transaction", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

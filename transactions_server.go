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
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	// Настройка формата логов
	logger.Formatter = &logrus.JSONFormatter{}
	// Установка минимального уровня логирования
	logger.Level = logrus.InfoLevel
	// Установка выходного потока
	logger.Out = os.Stdout
}

type Transaction struct {
	ID          string    `json:"id"`
	UserID      int       `json:"user_id"`
	Amount      float64   `json:"amount"`
	Currency    string    `json:"currency"`
	Type        string    `json:"type"` // income, expense, transfer
	Category    string    `json:"category"`
	Date        time.Time `json:"date"`
	Description string    `json:"description"`
}

var (
	transactions = make(map[string]Transaction)
	mutex        = &sync.Mutex{}
)

var db *pgxpool.Pool

func main() {
	var err error
	databaseUrl := os.Getenv("DATABASE_URL")
	db, err = setupDatabase(databaseUrl)
	if err != nil {
		logger.Fatalf("Failed to setup database: %v", err)
	}
	defer db.Close()

	http.HandleFunc("/transactions", handleTransactions)
	http.HandleFunc("/transactions/", handleTransactionByID)

	if err := createTables(db); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	logger.Println("Server starting on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
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

	// Проверка, что userID является числом
	if _, err := strconv.Atoi(userID); err != nil {
		http.Error(w, "User ID must be an integer", http.StatusBadRequest)
		return
	}

	rows, err := db.Query(context.Background(), "SELECT id, user_id, amount, currency, type, category, date, description FROM transactions WHERE user_id=$1", userID)
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

	// Проверка наличия пользователя
	var userID int
	err := db.QueryRow(context.Background(), "SELECT id FROM users WHERE id = $1", t.UserID).Scan(&userID)
	if err != nil {
		// Пользователь не найден, создаем нового пользователя
		username := "Default User"                           // Примерное имя, можно передавать в запросе или генерировать
		email := fmt.Sprintf("user%d@example.com", t.UserID) // Примерная почта, можно улучшить логику
		password := "example_password"                       // Пример пароля, в реальных условиях нужно хеширование

		_, err := db.Exec(context.Background(), "INSERT INTO users (name, email, password) VALUES ($1, $2, $3)", username, email, password)
		if err != nil {
			http.Error(w, "Failed to create user", http.StatusInternalServerError)
			return
		}
	}

	// Создание транзакции
	sql := `INSERT INTO transactions (user_id, amount, currency, type, category, date, description)
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

// Это простой способ сгенерировать уникальный ID для наших целей.
// В продакшене вы захотите использовать более надежный метод генерации уникальных ID.
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
		getTransactionByID(w, r, id) // Правильный вызов
	case "PUT":
		updateTransaction(w, r, id) // Правильный вызов
	case "DELETE":
		deleteTransaction(w, r, id) // Правильный вызов
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func getTransactionByID(w http.ResponseWriter, r *http.Request, id string) {
	var t Transaction
	err := db.QueryRow(context.Background(), "SELECT * FROM transactions WHERE id=$1", id).Scan(&t.ID, &t.UserID, &t.Amount, &t.Currency, &t.Type, &t.Category, &t.Date, &t.Description)
	if err != nil {
		http.Error(w, "Transaction not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(t)
}

func updateTransaction(w http.ResponseWriter, r *http.Request, id string) {
	var t Transaction
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sql := `UPDATE transactions SET amount=$1, currency=$2, type=$3, category=$4, date=$5, description=$6 WHERE id=$7`
	_, err := db.Exec(context.Background(), sql, t.Amount, t.Currency, t.Type, t.Category, t.Date, t.Description, id)
	if err != nil {
		http.Error(w, "Failed to update transaction", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func deleteTransaction(w http.ResponseWriter, r *http.Request, id string) {
	_, err := db.Exec(context.Background(), "DELETE FROM transactions WHERE id=$1", id)
	if err != nil {
		http.Error(w, "Failed to delete transaction", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

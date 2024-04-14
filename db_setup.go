package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

// Функция инициализации базы данных
func setupDatabase() (*pgxpool.Pool, error) {
	err := godotenv.Load()
	if err != nil {
		logger.WithError(err).Error("Failed to load .env file")
		return nil, fmt.Errorf("failed to load .env file: %v", err)
	}

	dbURL := os.Getenv("DATABASE_URL")
	ctx := context.Background()
	dbpool, err := pgxpool.Connect(ctx, dbURL)
	if err != nil {
		logger.WithError(err).Error("Unable to connect to database")
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	err = createTables(dbpool)
	if err != nil {
		logger.WithError(err).Error("Failed to create tables")
		return nil, err
	}

	logger.Info("Database setup completed successfully")
	return dbpool, nil
}

// Функция создания таблиц в базе данных
func createTables(db *pgxpool.Pool) error {
	ctx := context.Background()
	createTablesSQL := `
    CREATE TABLE IF NOT EXISTS users (
        id SERIAL PRIMARY KEY,
        name VARCHAR(255) NOT NULL,
        email VARCHAR(255) UNIQUE NOT NULL,
        password VARCHAR(255) NOT NULL
    );
    CREATE TABLE IF NOT EXISTS transactions (
        id SERIAL PRIMARY KEY,
        user_id INTEGER NOT NULL,
        amount DECIMAL(10, 2) NOT NULL,
        currency VARCHAR(3) NOT NULL,
        type VARCHAR(50) NOT NULL,
        category VARCHAR(50),
        date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
        description TEXT,
        FOREIGN KEY (user_id) REFERENCES users(id)
    );
	CREATE TABLE IF NOT EXISTS commissions (
    id SERIAL PRIMARY KEY,
    transaction_id INTEGER NOT NULL,
    amount DECIMAL(10, 2) NOT NULL,
    currency VARCHAR(3) NOT NULL,
    date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    description TEXT,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id)
);
    `
	_, err := db.Exec(ctx, createTablesSQL)
	if err != nil {
		logger.WithError(err).Error("Failed to create tables")
		return fmt.Errorf("failed to create tables: %v", err)
	}

	logger.Info("Tables created successfully")
	return nil
}

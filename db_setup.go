package main

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4/pgxpool"
)

// Функция инициализации базы данных
func setupDatabase(url string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	dbpool, err := pgxpool.Connect(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v", err)
	}

	err = createTables(dbpool)
	if err != nil {
		return nil, err
	}

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
    `
	_, err := db.Exec(ctx, createTablesSQL)
	if err != nil {
		return fmt.Errorf("failed to create tables: %v", err)
	}

	return nil
}

package db

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq" // Подключение драйвера PostgreSQL
)

var DB *sql.DB

// Connect открывает соединение с базой данных
func Connect() (*sql.DB, error) {
	connStr := os.Getenv("DATABASE_URL") // Путь к базе данных из .env
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Проверка соединения
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %v", err)
	}

	DB = db
	return db, nil
}

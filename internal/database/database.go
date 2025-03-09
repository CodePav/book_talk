package db

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

func ConnectDB() (*sql.DB, error) {
	connStr := "host=localhost port=5432 user=admin password=admin dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка подключения к базе данных: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ошибка при подключении к базе данных: %v", err)
	}

	log.Println("Подключение к базе данных установлено")
	return db, nil
}

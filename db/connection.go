package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func ConnectDB(host, user, password, dbname string) *sql.DB {
	connStr := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=5432 sslmode=disable",
		host, user, password, dbname,
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Gagal koneksi DB: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("❌ Tidak bisa ping DB: %v", err)
	}

	return db
}
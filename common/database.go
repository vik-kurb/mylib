package common

import (
	"database/sql"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func SetupDB(envPath string, testDBEnv string) (*sql.DB, error) {
	err := godotenv.Load(envPath)
	if err != nil {
		return nil, err
	}
	dbUrl := os.Getenv(testDBEnv)
	return sql.Open("postgres", dbUrl)
}

func CloseDB(db *sql.DB) {
	err := db.Close()
	if err != nil {
		log.Printf("Failed to close db: %v", err)
	}
}

func CloseRows(rows *sql.Rows) {
	err := rows.Close()
	if err != nil {
		log.Printf("Failed to close rows: %v", err)
	}
}

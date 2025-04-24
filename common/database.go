package common

import (
	"database/sql"
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

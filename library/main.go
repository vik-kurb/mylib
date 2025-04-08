package main

import (
	"database/sql"
	"fmt"
	"library/internal/database"
	"library/internal/server"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	env_err := godotenv.Load()
	if env_err != nil {
		fmt.Println("Error while loading env ", env_err)
		return
	}
	db_url := os.Getenv("DB_URL")

	db, sql_err := sql.Open("postgres", db_url)
	if sql_err != nil {
		fmt.Println("SQL error is: ", sql_err)
		return
	}

	sm := http.NewServeMux()
	apiCfg := server.ApiConfig{DB: database.New(db)}
	server.Handle(sm, &apiCfg)

	s := http.Server{
		Addr:    ":8080",
		Handler: sm,
	}
	err := s.ListenAndServe()
	if err != nil {
		fmt.Println("Error is: ", err)
		return
	}
}

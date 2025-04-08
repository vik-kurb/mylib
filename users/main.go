package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"users/internal/database"
	"users/internal/server"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	env_err := godotenv.Load()
	if env_err != nil {
		log.Fatal("Failed loading env ", env_err)
	}
	db_url := os.Getenv("DB_URL")

	db, sql_err := sql.Open("postgres", db_url)
	if sql_err != nil {
		log.Fatal("Failed SQL connect: ", sql_err)
	}

	sm := http.NewServeMux()
	apiCfg := server.ApiConfig{DB: database.New(db), AuthSecretKey: os.Getenv("AUTH_SECRET_KEY")}
	server.Handle(sm, &apiCfg)

	s := http.Server{
		Addr:    ":8080",
		Handler: sm,
	}
	err := s.ListenAndServe()
	if err != nil {
		log.Fatal("Failed starting server: ", err)
	}
}

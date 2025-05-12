package main

import (
	"log"
	"net/http"
	"os"
	"user-reading/internal/server"

	common "github.com/bakurvik/mylib-common"

	_ "github.com/bakurvik/mylib/user-reading/docs"

	_ "github.com/lib/pq"
)

// @title User reading Service API
// @version 1.0
// @description API for managing users' read books.

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

func main() {
	db, err := common.SetupDB("./.env")
	if err != nil {
		log.Fatal("Failed setup db ", err)
	}

	sm := http.NewServeMux()
	apiCfg := server.ApiConfig{DB: db, UsersServiceHost: os.Getenv("USERS_SERVICE_HOST"), UsersServicePort: os.Getenv("USERS_SERVICE_PORT")}
	server.Handle(sm, &apiCfg)

	s := http.Server{
		Addr:    ":8080",
		Handler: sm,
	}
	serverErr := s.ListenAndServe()
	if serverErr != nil {
		log.Fatal("Failed starting server: ", serverErr)
	}
}

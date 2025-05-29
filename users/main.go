package main

import (
	"log"
	"net/http"
	"os"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/users/internal/database"
	"github.com/bakurvik/mylib/users/internal/server"

	_ "github.com/bakurvik/mylib/users/docs"

	_ "github.com/lib/pq"
)

// @title Users Service API
// @version 1.0
// @description API for managing users data.

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
	apiCfg := server.ApiConfig{DB: database.New(db), AuthSecretKey: os.Getenv("AUTH_SECRET_KEY")}
	server.Handle(sm, &apiCfg)

	s := http.Server{
		Addr:    ":8080",
		Handler: common.CORSMiddleware(common.LoggingMiddleware(sm)),
	}
	serverErr := s.ListenAndServe()
	if serverErr != nil {
		log.Fatal("Failed starting server: ", serverErr)
	}
}

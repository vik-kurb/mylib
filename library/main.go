package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/library/internal/server"

	_ "github.com/bakurvik/mylib/library/docs"

	_ "github.com/lib/pq"
)

// @title Library Service API
// @version 1.0
// @description API for managing authors and books.

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /

const (
	defaultMaxSearchBooksLimit = 10
)

func main() {
	db, err := common.SetupDB("./.env")
	if err != nil {
		log.Fatal("Failed setup db ", err)
	}

	sm := http.NewServeMux()
	maxSearchBooksLimit, err := strconv.Atoi(os.Getenv("MAX_SEARCH_BOOKS_LIMIT"))
	if err != nil {
		log.Print("Invalid MAX_SEARCH_BOOKS_LIMIT value: ", os.Getenv("MAX_SEARCH_BOOKS_LIMIT"))
		maxSearchBooksLimit = defaultMaxSearchBooksLimit
	}
	apiCfg := server.ApiConfig{DB: db, MaxSearchBooksLimit: maxSearchBooksLimit}
	server.Handle(sm, &apiCfg)

	s := http.Server{
		Addr:    ":8080",
		Handler: common.LoggingMiddleware(sm),
	}
	serverErr := s.ListenAndServe()
	if serverErr != nil {
		log.Fatal("Failed starting server: ", serverErr)
	}
}

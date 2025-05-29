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
	defaultMaxSearchBooksLimit   = 10
	defaultMaxSearchAuthorsLimit = 10
)

func getLimit(varName string, defaultValue int) int {
	limit, err := strconv.Atoi(os.Getenv(varName))
	if err != nil {
		log.Printf("Invalid limit %v value: %v", varName, os.Getenv(varName))
		return defaultValue
	}
	return limit
}

func main() {
	db, err := common.SetupDB("./.env")
	if err != nil {
		log.Fatal("Failed setup db ", err)
	}

	sm := http.NewServeMux()
	apiCfg := server.ApiConfig{DB: db, MaxSearchBooksLimit: getLimit("MAX_SEARCH_BOOKS_LIMIT", defaultMaxSearchBooksLimit), MaxSearchAuthorsLimit: getLimit("MAX_SEARCH_AUTHORS_LIMIT", defaultMaxSearchAuthorsLimit)}
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

package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bakurvik/mylib/user-reading/internal/clients"
	"github.com/bakurvik/mylib/user-reading/internal/config"
	"github.com/bakurvik/mylib/user-reading/internal/server"

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

func getBooksCacheConfig() config.BooksCacheConfig {
	cfg := config.BooksCacheConfig{
		Enable:        false,
		CleanupPeriod: time.Hour,
	}

	useCache := os.Getenv("LIBRARY_BOOKS_CACHE_ENABLE")
	if useCache == "true" {
		cfg.Enable = true
	}

	period, err := time.ParseDuration(os.Getenv("LIBRARY_BOOKS_CACHE_CLEANUP_PERIOD_MIN"))
	if err != nil {
		log.Print("Invalid books cache cleanup period: ", period)
	} else {
		cfg.CleanupPeriod = period
	}

	threshold, err := time.ParseDuration(os.Getenv("LIBRARY_BOOKS_CACHE_CLEANUP_OLD_THRESHOLD_MIN"))
	if err != nil {
		log.Print("Invalid books cache cleanup threshold: ", threshold)
	} else {
		cfg.CleanupOldDataThreshold = threshold
	}
	return cfg
}

func main() {
	db, err := common.SetupDB("./.env")
	if err != nil {
		log.Fatal("Failed setup db ", err)
	}

	sm := http.NewServeMux()
	apiCfg := server.ApiConfig{DB: db, UsersServiceHost: os.Getenv("USERS_SERVICE_HOST"), LibraryServiceHost: os.Getenv("LIBRARY_SERVICE_HOST"), BooksCacheCfg: getBooksCacheConfig()}
	server.Handle(sm, &apiCfg)

	go clients.CleanupBooksCache(apiCfg.BooksCacheCfg)

	s := http.Server{
		Addr:    ":8080",
		Handler: common.CORSMiddleware(common.LoggingMiddleware(sm)),
	}
	serverErr := s.ListenAndServe()
	if serverErr != nil {
		log.Fatal("Failed starting server: ", serverErr)
	}
}

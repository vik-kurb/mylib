package server

import (
	"mylib/internal/database"
	"net/http"
)

type ApiConfig struct {
	DB *database.Queries
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	sm.HandleFunc("POST /api/authors", apiCfg.HandleApiAuthors)
}

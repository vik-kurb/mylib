package server

import (
	"net/http"
	"users/internal/database"
)

type ApiConfig struct {
	DB            *database.Queries
	AuthSecretKey string
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	sm.HandleFunc("POST /api/users", apiCfg.HandlePostApiUsers)
}

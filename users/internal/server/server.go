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
	sm.HandleFunc("POST /api/login", apiCfg.HandlePostApiLogin)
	sm.HandleFunc("POST /api/refresh", apiCfg.HandlePostApiRefresh)
	sm.HandleFunc("POST /api/revoke", apiCfg.HandlePostApiRevoke)
	sm.HandleFunc("PUT /api/users", apiCfg.HandlePutApiUsers)
	sm.HandleFunc("GET /api/users/{userID}", apiCfg.HandleGetApiUsers)
}

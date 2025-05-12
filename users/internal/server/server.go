package server

import (
	"fmt"
	"net/http"

	"github.com/bakurvik/mylib/users/internal/database"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	PingPath        = "/ping"
	ApiUsersPath    = "/api/users"
	AuthRevokePath  = "/auth/revoke"
	AuthLoginPath   = "/auth/login"
	AuthRefreshPath = "/auth/refresh"
	AuthWhoamiPath  = "/auth/whoami"
)

type ApiConfig struct {
	DB            *database.Queries
	AuthSecretKey string
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	// Ping
	sm.HandleFunc("GET "+PingPath, apiCfg.HandlePing)

	// Users
	sm.HandleFunc("POST "+ApiUsersPath, apiCfg.HandlePostApiUsers)
	sm.HandleFunc("PUT "+ApiUsersPath, apiCfg.HandlePutApiUsers)
	sm.HandleFunc(fmt.Sprintf("GET %v/{userID}", ApiUsersPath), apiCfg.HandleGetApiUsers)
	sm.HandleFunc("DELETE "+ApiUsersPath, apiCfg.HandleDeleteApiUsers)

	// Auth
	sm.HandleFunc("POST "+AuthLoginPath, apiCfg.HandlePostAuthLogin)
	sm.HandleFunc("POST "+AuthRefreshPath, apiCfg.HandlePostAuthRefresh)
	sm.HandleFunc("POST "+AuthRevokePath, apiCfg.HandlePostAuthRevoke)
	sm.HandleFunc("GET "+AuthWhoamiPath, apiCfg.HandleGetAuthWhoami)

	// Swagger
	sm.Handle("/swagger/", httpSwagger.WrapHandler)
}

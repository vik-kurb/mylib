package server

import (
	"database/sql"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	ApiUserReadingPath = "/api/user-reading"
	PingPath           = "/ping"
)

type ApiConfig struct {
	DB               *sql.DB
	UsersServiceHost string
	UsersServicePort int
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	// Ping
	sm.HandleFunc("GET "+PingPath, apiCfg.HandlePing)

	// User reading
	sm.HandleFunc("POST "+ApiUserReadingPath, apiCfg.HandlePostApiUserReadingPath)

	// Swagger
	sm.Handle("/swagger/", httpSwagger.WrapHandler)
}

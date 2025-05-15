package server

import (
	"database/sql"
	"fmt"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	ApiUserReadingPath = "/api/user-reading"
	PingPath           = "/ping"
)

type ApiConfig struct {
	DB                 *sql.DB
	UsersServiceHost   string
	LibraryServiceHost string
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	// Ping
	sm.HandleFunc("GET "+PingPath, apiCfg.HandlePing)

	// User reading
	sm.HandleFunc("POST "+ApiUserReadingPath, apiCfg.HandlePostApiUserReadingPath)
	sm.HandleFunc("PUT "+ApiUserReadingPath, apiCfg.HandlePutApiUserReadingPath)
	sm.HandleFunc(fmt.Sprintf("DELETE %v/{bookID}", ApiUserReadingPath), apiCfg.HandleDeleteApiUserReadingPath)

	// Swagger
	sm.Handle("/swagger/", httpSwagger.WrapHandler)
}

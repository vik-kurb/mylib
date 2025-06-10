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
	DB                   *sql.DB
	UsersServiceHost     string
	LibraryServiceHost   string
	UseLibraryBooksCache bool
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	// Ping
	sm.HandleFunc("GET "+PingPath, apiCfg.HandlePing)

	// User reading
	sm.HandleFunc("POST "+ApiUserReadingPath, apiCfg.HandlePostApiUserReadingPath)
	sm.HandleFunc("PUT "+ApiUserReadingPath, apiCfg.HandlePutApiUserReadingPath)
	sm.HandleFunc(fmt.Sprintf("DELETE %v/{bookID}", ApiUserReadingPath), apiCfg.HandleDeleteApiUserReadingPath)
	sm.HandleFunc("GET "+ApiUserReadingPath, apiCfg.HandleGetApiUserReadingPath)
	sm.HandleFunc(fmt.Sprintf("GET %v/{bookID}", ApiUserReadingPath), apiCfg.HandleGetApiUserReadingByBookPath)

	// Swagger
	sm.Handle("/swagger/", httpSwagger.WrapHandler)
}

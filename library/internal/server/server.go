package server

import (
	"database/sql"
	"fmt"
	"net/http"

	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	ApiAuthorsPath     = "/api/authors"
	AdminAuthorsPath   = "/admin/authors"
	ApiBooksPath       = "/api/books"
	ApiBooksSearchPath = "/api/books/search"
	AdminBooksPath     = "/admin/books"
	PingPath           = "/ping"
)

type ApiConfig struct {
	DB *sql.DB
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	// Ping
	sm.HandleFunc("GET "+PingPath, apiCfg.HandlePing)

	// Authors
	sm.HandleFunc("POST "+ApiAuthorsPath, apiCfg.HandlePostApiAuthors)
	sm.HandleFunc("GET "+ApiAuthorsPath, apiCfg.HandleGetApiAuthors)
	sm.HandleFunc(fmt.Sprintf("GET %v/{id}", ApiAuthorsPath), apiCfg.HandleGetApiAuthorsId)
	sm.HandleFunc(fmt.Sprintf("DELETE %v/{id}", AdminAuthorsPath), apiCfg.HandleDeleteAdminAuthors)
	sm.HandleFunc("PUT "+ApiAuthorsPath, apiCfg.HandlePutApiAuthors)
	sm.HandleFunc(fmt.Sprintf("GET %v/{id}/books", ApiAuthorsPath), apiCfg.HandleGetApiAuthorsBooks)

	// Books
	sm.HandleFunc("POST "+ApiBooksPath, apiCfg.HandlePostApiBooks)
	sm.HandleFunc("PUT "+ApiBooksPath, apiCfg.HandlePutApiBooks)
	sm.HandleFunc(fmt.Sprintf("DELETE %v/{id}", AdminBooksPath), apiCfg.HandleDeleteAdminBooks)
	sm.HandleFunc("POST "+ApiBooksSearchPath, apiCfg.HandlePostApiBooksSearch)

	// Swagger
	sm.Handle("/swagger/", httpSwagger.WrapHandler)
}

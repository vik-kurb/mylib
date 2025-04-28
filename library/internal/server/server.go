package server

import (
	"database/sql"
	"fmt"
	"net/http"
)

const (
	ApiAuthorsPath   = "/api/authors"
	AdminAuthorsPath = "/admin/authors"
	ApiBooksPath     = "/api/books"
)

type ApiConfig struct {
	DB *sql.DB
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
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
}

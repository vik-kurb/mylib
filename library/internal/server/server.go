package server

import (
	"fmt"
	"net/http"

	"github.com/bakurvik/mylib/library/internal/database"
)

const (
	ApiAuthorsPath   = "/api/authors"
	AdminAuthorsPath = "/admin/authors"
)

type ApiConfig struct {
	DB *database.Queries
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	sm.HandleFunc("POST "+ApiAuthorsPath, apiCfg.HandlePostApiAuthors)
	sm.HandleFunc("GET "+ApiAuthorsPath, apiCfg.HandleGetApiAuthors)
	sm.HandleFunc(fmt.Sprintf("GET %v/{id}", ApiAuthorsPath), apiCfg.HandleGetApiAuthorsId)
	sm.HandleFunc(fmt.Sprintf("DELETE %v/{id}", AdminAuthorsPath), apiCfg.HandleDeleteAdminAuthors)
	sm.HandleFunc("PUT "+ApiAuthorsPath, apiCfg.HandlePutApiAuthors)
}

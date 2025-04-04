package server

import (
	"mylib/internal/database"
	"net/http"
)

type ApiConfig struct {
	DB *database.Queries
}

func Handle(sm *http.ServeMux, apiCfg *ApiConfig) {
	sm.HandleFunc("POST /api/authors", apiCfg.HandlePostApiAuthors)
	sm.HandleFunc("GET /api/authors", apiCfg.HandleGetApiAuthors)
	sm.HandleFunc("GET /api/authors/{id}", apiCfg.HandleGetApiAuthorsId)
	sm.HandleFunc("DELETE /admin/authors/{id}", apiCfg.HandleDeleteAdminAuthors)
}

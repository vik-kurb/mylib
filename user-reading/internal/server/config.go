package server

import (
	"database/sql"

	"github.com/bakurvik/mylib/user-reading/internal/config"
)

type ApiConfig struct {
	DB                   *sql.DB
	UsersServiceHost     string
	LibraryServiceHost   string
	UseLibraryBooksCache bool
	BooksCacheCfg        config.BooksCacheConfig
}

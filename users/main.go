package main

import (
	"log"
	"net/http"
	"os"

	"github.com/bakurvik/mylib/common"
	"github.com/bakurvik/mylib/users/internal/database"
	"github.com/bakurvik/mylib/users/internal/server"

	_ "github.com/lib/pq"
)

func main() {
	db, err := common.SetupDB("./.env", "DB_URL")
	if err != nil {
		log.Fatal("Failed setup db ", err)
	}

	sm := http.NewServeMux()
	apiCfg := server.ApiConfig{DB: database.New(db), AuthSecretKey: os.Getenv("AUTH_SECRET_KEY")}
	server.Handle(sm, &apiCfg)

	s := http.Server{
		Addr:    ":8080",
		Handler: sm,
	}
	serverErr := s.ListenAndServe()
	if serverErr != nil {
		log.Fatal("Failed starting server: ", serverErr)
	}
}

package test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"user-reading/internal/server"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/user-reading/internal/database"
	"github.com/stretchr/testify/assert"
)

func setupTestServer(db *sql.DB) *httptest.Server {
	apiCfg := server.ApiConfig{DB: database.New(db)}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm)
}

func TestPing_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.PingPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

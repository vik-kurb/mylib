package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/user-reading/internal/clients"
	"github.com/bakurvik/mylib/user-reading/internal/server"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	deleteUserReading = "DELETE FROM user_reading"
	selectUserReading = "SELECT book_id, status FROM user_reading WHERE user_id = $1"
)

type userReading struct {
	bookID uuid.UUID
	status string
}

func mockUsersServer(t *testing.T, userID uuid.UUID, authHeader, authToken string) *httptest.Server {
	usersServiceMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == clients.UsersAuthWhoamiPath {
			assert.Equal(t, r.Header.Get(authHeader), authToken)
			w.WriteHeader(http.StatusOK)
			common.RespondWithJSON(w, http.StatusOK, clients.ResponseUserID{ID: userID.String()}, nil)
			return
		}
		http.NotFound(w, r)
	}))
	return usersServiceMock
}

func mockLibraryServer(t *testing.T, bookID uuid.UUID) *httptest.Server {
	libraryServiceMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("%v/%v", clients.LibraryApiBooksPath, bookID) {
			w.WriteHeader(http.StatusOK)
			common.RespondWithJSON(w, http.StatusOK, struct{}{}, nil)
		}
		http.NotFound(w, r)
	}))
	return libraryServiceMock
}

func setupTestServers(t *testing.T, db *sql.DB, userID uuid.UUID, authHeader string, authToken string, bookID uuid.UUID) (*httptest.Server, *httptest.Server, *httptest.Server) {
	usersServer := mockUsersServer(t, userID, authHeader, authToken)
	usersUrl, _ := url.Parse(usersServer.URL)
	libraryServer := mockLibraryServer(t, bookID)
	libraryUrl, _ := url.Parse(libraryServer.URL)

	apiCfg := server.ApiConfig{DB: db, UsersServiceHost: usersUrl.String(), LibraryServiceHost: libraryUrl.String()}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm), usersServer, libraryServer
}

func cleanupDB(db *sql.DB) {
	_, err := db.Query(deleteUserReading)
	if err != nil {
		log.Print("Failed to cleanup db: ", err)
	}
}

func getDbUserReading(t *testing.T, db *sql.DB, userID uuid.UUID) []userReading {
	rows, err := db.Query(selectUserReading, userID)
	if err != nil {
		t.Fatalf("Error while selecting user reading: %v", err)
	}
	defer common.CloseRows(rows)
	user_readings := make([]userReading, 0)

	for rows.Next() {
		ur := userReading{}
		err := rows.Scan(&ur.bookID, &ur.status)
		if err != nil {
			log.Fatal("Error scanning row:", err)
		}
		user_readings = append(user_readings, ur)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error reading rows:", err)
	}
	return user_readings
}

func TestPing_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)

	s, usersServer, libraryServer := setupTestServers(t, db, uuid.Nil, "", "", uuid.Nil)
	defer s.Close()
	defer usersServer.Close()
	defer libraryServer.Close()

	response, err := http.Get(s.URL + server.PingPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestCreateUserReading_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	userID := uuid.New()
	bookID := uuid.New()
	status := "finished"
	authHeader := "Authorization"
	authToken := "Bearer access_token"

	s, usersServer, libraryServer := setupTestServers(t, db, userID, authHeader, authToken, bookID)
	defer s.Close()
	defer usersServer.Close()
	defer libraryServer.Close()

	requestUserReading := server.RequestUserReading{BookId: bookID.String(), Status: status}
	body, _ := json.Marshal(requestUserReading)
	client := &http.Client{}
	request, err := http.NewRequest("POST", s.URL+server.ApiUserReadingPath, bytes.NewBuffer(body))
	assert.NoError(t, err)
	request.Header.Add(authHeader, authToken)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	b, _ := io.ReadAll(response.Body)
	responseData := server.ErrorResponse{}
	err = json.Unmarshal(b, &responseData)
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	userReadings := getDbUserReading(t, db, userID)

	assert.Equal(t, len(userReadings), 1)
	assert.Equal(t, userReadings[0].bookID, bookID)
	assert.Equal(t, userReadings[0].status, status)
}

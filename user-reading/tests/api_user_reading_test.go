package test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
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
	insertUserReading = "INSERT INTO user_reading(user_id, book_id, status) VALUES($1, $2, $3)"
)

type userReading struct {
	bookID uuid.UUID
	status string
}

type usersServiceData struct {
	userID     uuid.UUID
	authHeader string
	authToken  string
	statusCode int
}

type libraryServiceData struct {
	bookID     string
	statusCode int
}

func mockUsersServer(t *testing.T, data usersServiceData) *httptest.Server {
	usersServiceMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == clients.UsersAuthWhoamiPath {
			if data.authHeader != "" {
				assert.Equal(t, r.Header.Get(data.authHeader), data.authToken)
			}
			common.RespondWithJSON(w, data.statusCode, clients.ResponseUserID{ID: data.userID.String()}, nil)
			return
		}
		http.NotFound(w, r)
	}))
	return usersServiceMock
}

func mockLibraryServer(t *testing.T, data libraryServiceData) *httptest.Server {
	libraryServiceMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == fmt.Sprintf("%v/%v", clients.LibraryApiBooksPath, data.bookID) {
			common.RespondWithJSON(w, data.statusCode, struct{}{}, nil)
			return
		}
		http.NotFound(w, r)
	}))
	return libraryServiceMock
}

func setupTestServers(t *testing.T, db *sql.DB, usersData usersServiceData, libraryData libraryServiceData) (*httptest.Server, *httptest.Server, *httptest.Server) {
	usersServer := mockUsersServer(t, usersData)
	usersUrl, _ := url.Parse(usersServer.URL)
	libraryServer := mockLibraryServer(t, libraryData)
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

func getDBUserReading(t *testing.T, db *sql.DB, userID uuid.UUID) []userReading {
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

func addDBUserReading(db *sql.DB, userID string, userReadings []userReading) {
	for _, userReading := range userReadings {
		_, err := db.Exec(
			insertUserReading,
			userID, userReading.bookID, userReading.status)
		if err != nil {
			log.Print("Failed to add user reading to db: ", err)
		}
	}
}

func TestPing_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)

	s, usersServer, libraryServer := setupTestServers(t, db, usersServiceData{}, libraryServiceData{})
	defer s.Close()
	defer usersServer.Close()
	defer libraryServer.Close()

	response, err := http.Get(s.URL + server.PingPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestCreateUserReading(t *testing.T) {
	userID := uuid.New()
	bookID := uuid.New()

	type testCase struct {
		name                 string
		status               string
		usersData            usersServiceData
		libraryData          libraryServiceData
		expectedStatusCode   int
		expectedUserReadings []userReading
	}

	tests := []testCase{
		{
			name:                 "success",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			expectedStatusCode:   http.StatusCreated,
			expectedUserReadings: []userReading{{bookID: bookID, status: "finished"}},
		},
		{
			name:                 "invalid_book_id",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: "invalid_book_id", statusCode: http.StatusBadRequest},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []userReading{},
		},
		{
			name:                 "invalid_status",
			status:               "invalid_status",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []userReading{},
		},
		{
			name:                 "unauthorized",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusUnauthorized},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			expectedStatusCode:   http.StatusUnauthorized,
			expectedUserReadings: []userReading{},
		},
		{
			name:                 "book_not_found",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusNotFound},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []userReading{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			s, usersServer, libraryServer := setupTestServers(t, db, tc.usersData, tc.libraryData)
			defer s.Close()
			defer usersServer.Close()
			defer libraryServer.Close()

			requestUserReading := server.RequestUserReading{BookId: tc.libraryData.bookID, Status: tc.status}
			body, _ := json.Marshal(requestUserReading)
			client := &http.Client{}
			request, err := http.NewRequest("POST", s.URL+server.ApiUserReadingPath, bytes.NewBuffer(body))
			assert.NoError(t, err)
			request.Header.Add(tc.usersData.authHeader, tc.usersData.authToken)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			userReadings := getDBUserReading(t, db, userID)
			assert.ElementsMatch(t, userReadings, tc.expectedUserReadings)
		})
	}
}

func TestUpdateUserReading(t *testing.T) {
	userID := uuid.New()
	bookID := uuid.New()

	type testCase struct {
		name                 string
		status               string
		usersData            usersServiceData
		libraryData          libraryServiceData
		dbUserReadings       []userReading
		expectedStatusCode   int
		expectedUserReadings []userReading
	}

	tests := []testCase{
		{
			name:                 "success",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []userReading{{bookID: bookID, status: "want_to_read"}},
			expectedStatusCode:   http.StatusNoContent,
			expectedUserReadings: []userReading{{bookID: bookID, status: "finished"}},
		},
		{
			name:                 "invalid_book_id",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: "invalid_book_id", statusCode: http.StatusBadRequest},
			dbUserReadings:       []userReading{{bookID: bookID, status: "want_to_read"}},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []userReading{{bookID: bookID, status: "want_to_read"}},
		},
		{
			name:                 "invalid_status",
			status:               "invalid_status",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []userReading{{bookID: bookID, status: "want_to_read"}},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []userReading{{bookID: bookID, status: "want_to_read"}},
		},
		{
			name:                 "unauthorized",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusUnauthorized},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []userReading{{bookID: bookID, status: "want_to_read"}},
			expectedStatusCode:   http.StatusUnauthorized,
			expectedUserReadings: []userReading{{bookID: bookID, status: "want_to_read"}},
		},
		{
			name:                 "book_not_found",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusNotFound},
			dbUserReadings:       []userReading{{bookID: bookID, status: "want_to_read"}},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []userReading{{bookID: bookID, status: "want_to_read"}},
		},
		{
			name:                 "no_user_reading",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []userReading{},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []userReading{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			addDBUserReading(db, userID.String(), tc.dbUserReadings)

			s, usersServer, libraryServer := setupTestServers(t, db, tc.usersData, tc.libraryData)
			defer s.Close()
			defer usersServer.Close()
			defer libraryServer.Close()

			requestUserReading := server.RequestUserReading{BookId: tc.libraryData.bookID, Status: tc.status}
			body, _ := json.Marshal(requestUserReading)
			client := &http.Client{}
			request, err := http.NewRequest("PUT", s.URL+server.ApiUserReadingPath, bytes.NewBuffer(body))
			assert.NoError(t, err)
			request.Header.Add(tc.usersData.authHeader, tc.usersData.authToken)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			userReadings := getDBUserReading(t, db, userID)
			assert.ElementsMatch(t, userReadings, tc.expectedUserReadings)
		})
	}
}

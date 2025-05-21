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

type usersServiceData struct {
	userID     uuid.UUID
	authHeader string
	authToken  string
	statusCode int
}

type libraryServiceData struct {
	bookID     string
	statusCode int
	booksInfo  []clients.ResponseBookFullInfo
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
		if r.URL.Path == clients.LibraryApiBooksSearchPath {
			common.RespondWithJSON(w, data.statusCode, data.booksInfo, nil)
			return
		}
		http.NotFound(w, r)
	}))
	return libraryServiceMock
}

func setupTestServers(t *testing.T, db *sql.DB, usersData usersServiceData, libraryData libraryServiceData) (*httptest.Server, *httptest.Server, *httptest.Server) {
	usersServer := mockUsersServer(t, usersData)
	usersURL, _ := url.Parse(usersServer.URL)
	libraryServer := mockLibraryServer(t, libraryData)
	libraryURL, _ := url.Parse(libraryServer.URL)

	apiCfg := server.ApiConfig{DB: db, UsersServiceHost: usersURL.String(), LibraryServiceHost: libraryURL.String()}
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

func getDBUserReading(t *testing.T, db *sql.DB, userID uuid.UUID) []server.UserReading {
	rows, err := db.Query(selectUserReading, userID)
	if err != nil {
		t.Fatalf("Error while selecting user reading: %v", err)
	}
	defer common.CloseRows(rows)
	user_readings := make([]server.UserReading, 0)

	for rows.Next() {
		ur := server.UserReading{}
		err := rows.Scan(&ur.BookID, &ur.Status)
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

func addDBUserReading(db *sql.DB, userID string, userReadings []server.UserReading) {
	for _, userReading := range userReadings {
		_, err := db.Exec(
			insertUserReading,
			userID, userReading.BookID, userReading.Status)
		if err != nil {
			log.Print("Failed to add user reading to db: ", err)
		}
	}
}

func TestPing_Success(t *testing.T) {
	db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
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
		expectedUserReadings []server.UserReading
	}

	tests := []testCase{
		{
			name:                 "success",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			expectedStatusCode:   http.StatusCreated,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "finished"}},
		},
		{
			name:                 "invalid_book_id",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: "invalid_book_id", statusCode: http.StatusBadRequest},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{},
		},
		{
			name:                 "invalid_status",
			status:               "invalid_status",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{},
		},
		{
			name:                 "unauthorized",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusUnauthorized},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			expectedStatusCode:   http.StatusUnauthorized,
			expectedUserReadings: []server.UserReading{},
		},
		{
			name:                 "book_not_found",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusNotFound},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			s, usersServer, libraryServer := setupTestServers(t, db, tc.usersData, tc.libraryData)
			defer s.Close()
			defer usersServer.Close()
			defer libraryServer.Close()

			requestUserReading := server.UserReading{BookID: tc.libraryData.bookID, Status: tc.status}
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
		dbUserReadings       []server.UserReading
		expectedStatusCode   int
		expectedUserReadings []server.UserReading
	}

	tests := []testCase{
		{
			name:                 "success",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusNoContent,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "finished"}},
		},
		{
			name:                 "invalid_book_id",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: "invalid_book_id", statusCode: http.StatusBadRequest},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
		},
		{
			name:                 "invalid_status",
			status:               "invalid_status",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
		},
		{
			name:                 "unauthorized",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusUnauthorized},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusUnauthorized,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
		},
		{
			name:                 "book_not_found",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusNotFound},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
		},
		{
			name:                 "no_user_reading",
			status:               "finished",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:          libraryServiceData{bookID: bookID.String(), statusCode: http.StatusOK},
			dbUserReadings:       []server.UserReading{},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			addDBUserReading(db, userID.String(), tc.dbUserReadings)

			s, usersServer, libraryServer := setupTestServers(t, db, tc.usersData, tc.libraryData)
			defer s.Close()
			defer usersServer.Close()
			defer libraryServer.Close()

			requestUserReading := server.UserReading{BookID: tc.libraryData.bookID, Status: tc.status}
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

func TestDeleteUserReading(t *testing.T) {
	userID := uuid.New()
	bookID := uuid.New()

	type testCase struct {
		name                 string
		bookID               string
		usersData            usersServiceData
		dbUserReadings       []server.UserReading
		expectedStatusCode   int
		expectedUserReadings []server.UserReading
	}

	tests := []testCase{
		{
			name:                 "success",
			bookID:               bookID.String(),
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusNoContent,
			expectedUserReadings: []server.UserReading{},
		},
		{
			name:                 "invalid_book_id",
			bookID:               "invalid_book_id",
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusBadRequest,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
		},
		{
			name:                 "unauthorized",
			bookID:               bookID.String(),
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusUnauthorized},
			dbUserReadings:       []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
			expectedStatusCode:   http.StatusUnauthorized,
			expectedUserReadings: []server.UserReading{{BookID: bookID.String(), Status: "want_to_read"}},
		},
		{
			name:                 "no_user_reading",
			bookID:               bookID.String(),
			usersData:            usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			dbUserReadings:       []server.UserReading{},
			expectedStatusCode:   http.StatusNoContent,
			expectedUserReadings: []server.UserReading{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			addDBUserReading(db, userID.String(), tc.dbUserReadings)

			s, usersServer, libraryServer := setupTestServers(t, db, tc.usersData, libraryServiceData{})
			defer s.Close()
			defer usersServer.Close()
			defer libraryServer.Close()

			client := &http.Client{}
			request, err := http.NewRequest("DELETE", s.URL+server.ApiUserReadingPath+"/"+tc.bookID, nil)
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

func TestGetUserReading(t *testing.T) {
	userID := uuid.New()
	book1 := uuid.New()
	book2 := uuid.New()

	type testCase struct {
		name               string
		usersData          usersServiceData
		libraryData        libraryServiceData
		dbUserReadings     []server.UserReading
		expectedStatusCode int
		expectedResponse   []server.ResponseUserReading
	}

	tests := []testCase{
		{
			name:               "success_one_book",
			usersData:          usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:        libraryServiceData{statusCode: http.StatusOK, booksInfo: []clients.ResponseBookFullInfo{{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}}}},
			dbUserReadings:     []server.UserReading{{BookID: book1.String(), Status: "finished"}},
			expectedStatusCode: http.StatusOK,
			expectedResponse: []server.ResponseUserReading{
				{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}, Status: "finished"}},
		},
		{
			name:               "success_several_books_and_authors",
			usersData:          usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:        libraryServiceData{statusCode: http.StatusOK, booksInfo: []clients.ResponseBookFullInfo{{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}}, {ID: book2.String(), Title: "Title 2", Authors: []string{"Author 1", "Author 2"}}}},
			dbUserReadings:     []server.UserReading{{BookID: book1.String(), Status: "finished"}, {BookID: book2.String(), Status: "reading"}},
			expectedStatusCode: http.StatusOK,
			expectedResponse: []server.ResponseUserReading{
				{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}, Status: "finished"},
				{ID: book2.String(), Title: "Title 2", Authors: []string{"Author 1", "Author 2"}, Status: "reading"}},
		},
		{
			name:               "unauthorized",
			usersData:          usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusUnauthorized},
			libraryData:        libraryServiceData{statusCode: http.StatusOK, booksInfo: []clients.ResponseBookFullInfo{{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}}}},
			dbUserReadings:     []server.UserReading{{BookID: book1.String(), Status: "finished"}},
			expectedStatusCode: http.StatusUnauthorized,
			expectedResponse:   nil,
		},
		{
			name:               "no_user_reading",
			usersData:          usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:        libraryServiceData{statusCode: http.StatusOK, booksInfo: []clients.ResponseBookFullInfo{{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}}}},
			dbUserReadings:     []server.UserReading{},
			expectedStatusCode: http.StatusOK,
			expectedResponse:   []server.ResponseUserReading{},
		},
		{
			name:               "filter_out_unknown_book",
			usersData:          usersServiceData{userID: userID, authHeader: "Authorization", authToken: "Bearer access_token", statusCode: http.StatusOK},
			libraryData:        libraryServiceData{statusCode: http.StatusOK, booksInfo: []clients.ResponseBookFullInfo{{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}}}},
			dbUserReadings:     []server.UserReading{{BookID: book1.String(), Status: "finished"}, {BookID: book2.String(), Status: "reading"}},
			expectedStatusCode: http.StatusOK,
			expectedResponse: []server.ResponseUserReading{
				{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}, Status: "finished"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			addDBUserReading(db, userID.String(), tc.dbUserReadings)

			s, usersServer, libraryServer := setupTestServers(t, db, tc.usersData, tc.libraryData)
			defer s.Close()
			defer usersServer.Close()
			defer libraryServer.Close()

			client := &http.Client{}
			request, err := http.NewRequest("GET", s.URL+server.ApiUserReadingPath, nil)
			assert.NoError(t, err)
			request.Header.Add(tc.usersData.authHeader, tc.usersData.authToken)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.expectedResponse != nil {
				decoder := json.NewDecoder(response.Body)
				responseBody := []server.ResponseUserReading{}
				err = decoder.Decode(&responseBody)
				assert.NoError(t, err)

				assert.ElementsMatch(t, responseBody, tc.expectedResponse)
			}
		})
	}
}

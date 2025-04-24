package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bakurvik/mylib/common"
	"github.com/bakurvik/mylib/library/internal/database"
	"github.com/bakurvik/mylib/library/internal/server"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	selectAuthors = "SELECT full_name, birth_date, death_date, created_at, updated_at FROM authors"
	deleteAuthors = "TRUNCATE authors"
	insertAuthor  = "INSERT INTO authors(id, full_name, birth_date, death_date, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"
)

type Author struct {
	id        string
	fullName  string
	birthDate sql.NullTime
	deathDate sql.NullTime
	createdAt time.Time
	updatedAt time.Time
}

func assertDateEqual(t *testing.T, timeDate sql.NullTime, expectedData map[string]string, key string) {
	expectedDate, ok := expectedData[key]
	assert.Equal(t, ok, timeDate.Valid)
	if ok {
		assert.Equal(t, timeDate.Time.Format(common.DateFormat), expectedDate)
	}
}

func assertEqual(t *testing.T, author Author, expectedData map[string]string) {
	assert.Equal(t, author.fullName, expectedData["full_name"])
	assertDateEqual(t, author.birthDate, expectedData, "birth_date")
	assertDateEqual(t, author.deathDate, expectedData, "death_date")
}

func cleanupDB(db *sql.DB) {
	db.Query(deleteAuthors)
}

func AddAuthorsDB(db *sql.DB, authors []Author) {
	for _, author := range authors {
		db.Exec(
			insertAuthor,
			author.id, author.fullName, author.birthDate, author.deathDate, author.createdAt, author.updatedAt)
	}
}

func setupTestServer(db *sql.DB) *httptest.Server {
	apiCfg := server.ApiConfig{DB: database.New(db)}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm)
}

func GetDbAuthors(t *testing.T, db *sql.DB) []Author {
	rows, err := db.Query(selectAuthors)
	if err != nil {
		t.Fatalf("Error while selecting authors: %v", err)
	}
	defer rows.Close()
	authors := make([]Author, 0)

	for rows.Next() {
		a := Author{}
		err := rows.Scan(&a.fullName, &a.birthDate, &a.deathDate, &a.createdAt, &a.updatedAt)
		if err != nil {
			log.Fatal("Error scanning row:", err)
		}
		authors = append(authors, a)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error reading rows:", err)
	}
	return authors
}

func TestCreateAuthor_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	request := map[string]string{
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(request)

	response, err := http.Post(s.URL+server.ApiAuthorsPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], request)
}

func TestCreateAuthor_EmptyDates(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	request := map[string]string{
		"full_name": "Leo Tolstoy",
	}
	body, _ := json.Marshal(request)

	response, err := http.Post(s.URL+server.ApiAuthorsPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], request)
}

func TestCreateAuthor_BadRequest(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Post(s.URL+server.ApiAuthorsPath, "application/json", bytes.NewBuffer([]byte("{}")))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestGetAuthors_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	dbAuthors := []Author{
		{id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", fullName: "Alexander Pushkin"},
		{id: "0254a622-68fd-4812-a0bc-d997bbe3a731", fullName: "Leo Tolstoy"},
		{id: "1a05bda1-266c-4226-a662-51cbc60ddc86", fullName: "Fyodor Dostoevsky"},
	}
	AddAuthorsDB(db, dbAuthors)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.ApiAuthorsPath)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := make([]server.ResponseAuthorShortInfo, 0)
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, responseBody, []server.ResponseAuthorShortInfo{
		{FullName: "Alexander Pushkin", Id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea"},
		{FullName: "Fyodor Dostoevsky", Id: "1a05bda1-266c-4226-a662-51cbc60ddc86"},
		{FullName: "Leo Tolstoy", Id: "0254a622-68fd-4812-a0bc-d997bbe3a731"}})
}

func TestGetAuthors_NotFound(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.ApiAuthorsPath)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGetAuthorsId_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	author := Author{
		id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.20.1837"),
	}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiAuthorsPath, author.id))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := server.ResponseAuthorFullInfo{}
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, responseBody, server.ResponseAuthorFullInfo{FullName: author.fullName, BirthDate: author.birthDate.Time.Format(common.DateFormat), DeathDate: author.deathDate.Time.Format(common.DateFormat)})
}

func TestGetAuthorsId_NotFound(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	id := "aeecbc4e-9547-4fce-88ac-4e739567a1ea"
	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiAuthorsPath, id))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGetAuthorsId_InvalidId(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	id := "invalid_uuid"
	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiAuthorsPath, id))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestDeleteAuthorsId_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	author := Author{
		id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.20.1837"),
	}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminAuthorsPath, author.id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestDeleteAuthorsId_NoAuthor(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	id := "aeecbc4e-9547-4fce-88ac-4e739567a1ea"
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminAuthorsPath, id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestDeleteAuthorsId_InvalidId(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	id := "invalud id"
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminAuthorsPath, id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestUpdateAuthor_Success(t *testing.T) {
	db, dbErr := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, dbErr)
	defer db.Close()
	cleanupDB(db)
	author := Author{
		id:        "aeecbc4e-9547-4fce-88ac-4e739567a1ea",
		fullName:  "Alexander Pushkin",
		birthDate: common.ToNullTime("06.06.1799"),
		deathDate: common.ToNullTime("10.20.1837"),
		createdAt: time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
		updatedAt: time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC),
	}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	requestBody := map[string]string{
		"id":         author.id,
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(requestBody)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiAuthorsPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], requestBody)
	assert.True(t, authors[0].createdAt.Equal(author.createdAt))
	assert.False(t, authors[0].updatedAt.Equal(author.updatedAt))
}

func TestUpdateAuthor_NotFoundAuthor(t *testing.T) {
	db, dbErr := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, dbErr)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	requestBody := map[string]string{
		"id":         "aeecbc4e-9547-4fce-88ac-4e739567a1ea",
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(requestBody)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiAuthorsPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 0)
}

func TestUpdateAuthor_InvalidId(t *testing.T) {
	db, dbErr := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, dbErr)
	defer db.Close()
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	requestBody := map[string]string{
		"id":         "invalid_id",
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(requestBody)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiAuthorsPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 0)
}

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

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/library/internal/server"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	selectAuthors = "SELECT full_name, birth_date, death_date, created_at, updated_at FROM authors"
	insertAuthor  = "INSERT INTO authors(id, full_name, birth_date, death_date, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"
)

type Author struct {
	id        uuid.UUID
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

func AddAuthorsDB(db *sql.DB, authors []Author) {
	for _, author := range authors {
		_, err := db.Exec(
			insertAuthor,
			author.id, author.fullName, author.birthDate, author.deathDate, author.createdAt, author.updatedAt)
		if err != nil {
			log.Print("Failed to add author to db: ", err)
		}
	}
}

func setupTestServer(db *sql.DB) *httptest.Server {
	apiCfg := server.ApiConfig{DB: db}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm)
}

func GetDbAuthors(t *testing.T, db *sql.DB) []Author {
	rows, err := db.Query(selectAuthors)
	if err != nil {
		t.Fatalf("Error while selecting authors: %v", err)
	}
	defer common.CloseRows(rows)
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
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
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
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], request)
}

func TestCreateAuthor_EmptyDates(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	request := map[string]string{
		"full_name": "Leo Tolstoy",
	}
	body, _ := json.Marshal(request)

	response, err := http.Post(s.URL+server.ApiAuthorsPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], request)
}

func TestCreateAuthor_BadRequest(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Post(s.URL+server.ApiAuthorsPath, "application/json", bytes.NewBuffer([]byte("{}")))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestGetAuthors_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	dbAuthors := []Author{
		{id: uuid.New(), fullName: "Alexander Pushkin"},
		{id: uuid.New(), fullName: "Leo Tolstoy"},
		{id: uuid.New(), fullName: "Fyodor Dostoevsky"},
	}
	AddAuthorsDB(db, dbAuthors)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.ApiAuthorsPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := make([]server.ResponseAuthorShortInfo, 0)
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, responseBody, []server.ResponseAuthorShortInfo{
		{FullName: "Alexander Pushkin", Id: dbAuthors[0].id.String()},
		{FullName: "Fyodor Dostoevsky", Id: dbAuthors[2].id.String()},
		{FullName: "Leo Tolstoy", Id: dbAuthors[1].id.String()}})
}

func TestGetAuthors_NoAuthors(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.ApiAuthorsPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := make([]server.ResponseAuthorShortInfo, 0)
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, len(responseBody), 0)
}

func TestGetAuthorsId_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	author := Author{
		id: uuid.New(), fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837"),
	}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiAuthorsPath, author.id))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := server.ResponseAuthorFullInfo{}
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, responseBody, server.ResponseAuthorFullInfo{FullName: author.fullName, BirthDate: author.birthDate.Time.Format(common.DateFormat), DeathDate: author.deathDate.Time.Format(common.DateFormat)})
}

func TestGetAuthorsId_NotFound(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	id := uuid.New()
	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiAuthorsPath, id))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGetAuthorsId_InvalidId(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	id := "invalid_uuid"
	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiAuthorsPath, id))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestDeleteAuthorsId_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	author := Author{
		id: uuid.New(), fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837"),
	}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminAuthorsPath, author.id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestDeleteAuthorsId_NoAuthor(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	id := uuid.New()
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminAuthorsPath, id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestDeleteAuthorsId_InvalidId(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	id := "invalud id"
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminAuthorsPath, id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestUpdateAuthor_Success(t *testing.T) {
	db, dbErr := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, dbErr)
	defer common.CloseDB(db)
	cleanupDB(db)
	author := Author{
		id:        uuid.New(),
		fullName:  "Alexander Pushkin",
		birthDate: common.ToNullTime("06.06.1799"),
		deathDate: common.ToNullTime("10.02.1837"),
		createdAt: time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
		updatedAt: time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC),
	}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	requestBody := map[string]string{
		"id":         author.id.String(),
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
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], requestBody)
	assert.True(t, authors[0].createdAt.Equal(author.createdAt))
	assert.False(t, authors[0].updatedAt.Equal(author.updatedAt))
}

func TestUpdateAuthor_NotFoundAuthor(t *testing.T) {
	db, dbErr := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, dbErr)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	requestBody := map[string]string{
		"id":         uuid.New().String(),
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
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 0)
}

func TestUpdateAuthor_InvalidId(t *testing.T) {
	db, dbErr := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, dbErr)
	defer common.CloseDB(db)
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
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 0)
}

func TestGetAuthorBooks_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	authors := []Author{
		{id: uuid.New(), fullName: "Alexander Pushkin"},
		{id: uuid.New(), fullName: "Leo Tolstoy"},
	}
	AddAuthorsDB(db, authors)
	books := []Book{
		{id: uuid.New(), title: "The Captain's Daughter"},
		{id: uuid.New(), title: "Dubrovsky"},
		{id: uuid.New(), title: "War and Peace"},
	}
	AddBooksDB(db, books)
	AddBookAuthorsDB(db, books[0].id.String(), []string{authors[0].id.String()})
	AddBookAuthorsDB(db, books[1].id.String(), []string{authors[0].id.String()})
	AddBookAuthorsDB(db, books[2].id.String(), []string{authors[1].id.String()})

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}/books", s.URL, server.ApiAuthorsPath, authors[0].id))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := make([]server.ResponseBook, 0)
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, responseBody, []server.ResponseBook{{Id: books[1].id.String(), Title: books[1].title}, {Id: books[0].id.String(), Title: books[0].title}})
}

func TestGetAuthorBooks_NoBooks(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	authors := []Author{
		{id: uuid.New(), fullName: "Alexander Pushkin"},
		{id: uuid.New(), fullName: "Leo Tolstoy"},
	}
	AddAuthorsDB(db, authors)
	books := []Book{
		{id: uuid.New(), title: "War and Peace"},
	}
	AddBooksDB(db, books)
	AddBookAuthorsDB(db, books[0].id.String(), []string{authors[1].id.String()})

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}/books", s.URL, server.ApiAuthorsPath, authors[0].id))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := make([]server.ResponseBook, 0)
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, responseBody, []server.ResponseBook{})
}

func TestGetAuthorBooks_UnknownAuthor(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}/books", s.URL, server.ApiAuthorsPath, uuid.New()))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
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

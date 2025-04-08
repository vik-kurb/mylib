package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"library/internal/database"
	"library/internal/server"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	kApiAuthorsPath   = "/api/authors"
	kAdminAuthorsPath = "/admin/authors"
	kSelectAuthors    = "SELECT full_name, birth_date, death_date, created_at, updated_at FROM authors"
	kDeleteAuthors    = "TRUNCATE authors"
	kTimeFormat       = "02.01.2006"
)

type Author struct {
	id         string
	full_name  string
	birth_date sql.NullTime
	death_date sql.NullTime
	created_at time.Time
	updated_at time.Time
}

func assertDateEqual(t *testing.T, time_date sql.NullTime, expected_data map[string]string, key string) {
	expected_date, ok := expected_data[key]
	assert.Equal(t, ok, time_date.Valid)
	if ok {
		assert.Equal(t, time_date.Time.Format(kTimeFormat), expected_date)
	}
}

func assertEqual(t *testing.T, author Author, expected_data map[string]string) {
	assert.Equal(t, author.full_name, expected_data["full_name"])
	assertDateEqual(t, author.birth_date, expected_data, "birth_date")
	assertDateEqual(t, author.death_date, expected_data, "death_date")
}

func setupTestDB() (*sql.DB, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		return nil, err
	}
	db_url := os.Getenv("TEST_DB_URL")
	return sql.Open("postgres", db_url)
}

func cleanupDB(db *sql.DB) {
	db.Query(kDeleteAuthors)
}

func AddAuthorsDB(db *sql.DB, authors []Author) {
	for _, author := range authors {
		db.Exec(
			"INSERT INTO authors(id, full_name, birth_date, death_date, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
			author.id, author.full_name, author.birth_date, author.death_date, author.created_at, author.updated_at)
	}
}

func setupTestServer(db *sql.DB) *httptest.Server {
	apiCfg := server.ApiConfig{DB: database.New(db)}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm)
}

func GetDbAuthors(t *testing.T, db *sql.DB) []Author {
	rows, err := db.Query(kSelectAuthors)
	if err != nil {
		t.Fatalf("Error while selecting authors: %v", err)
	}
	defer rows.Close()
	authors := make([]Author, 0)

	for rows.Next() {
		a := Author{}
		err := rows.Scan(&a.full_name, &a.birth_date, &a.death_date, &a.created_at, &a.updated_at)
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

func toSqlNullTime(s string) sql.NullTime {
	t, err := time.Parse(kTimeFormat, s)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func TestCreateAuthor_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	request := map[string]string{
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(request)

	response, err := http.Post(server.URL+kApiAuthorsPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], request)
}

func TestCreateAuthor_EmptyDates(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	request := map[string]string{
		"full_name": "Leo Tolstoy",
	}
	body, _ := json.Marshal(request)

	response, err := http.Post(server.URL+kApiAuthorsPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], request)
}

func TestCreateAuthor_BadRequest(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	response, err := http.Post(server.URL+kApiAuthorsPath, "application/json", bytes.NewBuffer([]byte("{}")))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestGetAuthors_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	db_authors := []Author{
		{id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", full_name: "Alexander Pushkin"},
		{id: "0254a622-68fd-4812-a0bc-d997bbe3a731", full_name: "Leo Tolstoy"},
		{id: "1a05bda1-266c-4226-a662-51cbc60ddc86", full_name: "Fyodor Dostoevsky"},
	}
	AddAuthorsDB(db, db_authors)

	server := setupTestServer(db)
	defer server.Close()

	response, err := http.Get(server.URL + kApiAuthorsPath)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	type responseAuthor struct {
		FullName string `json:"full_name"`
		Id       string `json:"id"`
	}
	response_body := make([]responseAuthor, 0)
	err = decoder.Decode(&response_body)
	assert.NoError(t, err)
	assert.Equal(t, response_body, []responseAuthor{
		{FullName: "Alexander Pushkin", Id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea"},
		{FullName: "Fyodor Dostoevsky", Id: "1a05bda1-266c-4226-a662-51cbc60ddc86"},
		{FullName: "Leo Tolstoy", Id: "0254a622-68fd-4812-a0bc-d997bbe3a731"}})
}

func TestGetAuthors_NotFound(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	response, err := http.Get(server.URL + kApiAuthorsPath)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGetAuthorsId_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	author := Author{
		id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", full_name: "Alexander Pushkin", birth_date: toSqlNullTime("06.06.1799"), death_date: toSqlNullTime("10.20.1837"),
	}
	AddAuthorsDB(db, []Author{author})

	server := setupTestServer(db)
	defer server.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", server.URL, kApiAuthorsPath, author.id))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	type responseAuthor struct {
		FullName  string `json:"full_name"`
		BirthDate string `json:"birth_date,omitempty"`
		DeathDate string `json:"death_date,omitempty"`
	}
	response_body := responseAuthor{}
	err = decoder.Decode(&response_body)
	assert.NoError(t, err)
	assert.Equal(t, response_body, responseAuthor{FullName: author.full_name, BirthDate: author.birth_date.Time.Format(kTimeFormat), DeathDate: author.death_date.Time.Format(kTimeFormat)})
}

func TestGetAuthorsId_NotFound(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	id := "aeecbc4e-9547-4fce-88ac-4e739567a1ea"
	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", server.URL, kApiAuthorsPath, id))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGetAuthorsId_InvalidId(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	id := "invalid_uuid"
	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", server.URL, kApiAuthorsPath, id))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestDeleteAuthorsId_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	author := Author{
		id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", full_name: "Alexander Pushkin", birth_date: toSqlNullTime("06.06.1799"), death_date: toSqlNullTime("10.20.1837"),
	}
	AddAuthorsDB(db, []Author{author})

	server := setupTestServer(db)
	defer server.Close()

	client := &http.Client{}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", server.URL, kAdminAuthorsPath, author.id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestDeleteAuthorsId_NoAuthor(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	client := &http.Client{}
	id := "aeecbc4e-9547-4fce-88ac-4e739567a1ea"
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", server.URL, kAdminAuthorsPath, id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestDeleteAuthorsId_InvalidId(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	client := &http.Client{}
	id := "invalud id"
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", server.URL, kAdminAuthorsPath, id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestUpdateAuthor_Success(t *testing.T) {
	db, db_err := setupTestDB()
	assert.NoError(t, db_err)
	defer db.Close()
	cleanupDB(db)
	author := Author{
		id:         "aeecbc4e-9547-4fce-88ac-4e739567a1ea",
		full_name:  "Alexander Pushkin",
		birth_date: toSqlNullTime("06.06.1799"),
		death_date: toSqlNullTime("10.20.1837"),
		created_at: time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
		updated_at: time.Date(2025, 2, 5, 0, 0, 0, 0, time.UTC),
	}
	AddAuthorsDB(db, []Author{author})

	server := setupTestServer(db)
	defer server.Close()

	request_body := map[string]string{
		"id":         author.id,
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(request_body)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", server.URL, kApiAuthorsPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 1)
	assertEqual(t, authors[0], request_body)
	assert.True(t, authors[0].created_at.Equal(author.created_at))
	assert.False(t, authors[0].updated_at.Equal(author.updated_at))
}

func TestUpdateAuthor_NotFoundAuthor(t *testing.T) {
	db, db_err := setupTestDB()
	assert.NoError(t, db_err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	request_body := map[string]string{
		"id":         "aeecbc4e-9547-4fce-88ac-4e739567a1ea",
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(request_body)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", server.URL, kApiAuthorsPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 0)
}

func TestUpdateAuthor_InvalidId(t *testing.T) {
	db, db_err := setupTestDB()
	assert.NoError(t, db_err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	request_body := map[string]string{
		"id":         "invalid_id",
		"full_name":  "Leo Tolstoy",
		"birth_date": "09.09.1828",
		"death_date": "20.11.1910",
	}
	body, _ := json.Marshal(request_body)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", server.URL, kApiAuthorsPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)

	authors := GetDbAuthors(t, db)
	assert.Equal(t, len(authors), 0)
}

package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"mylib/internal/database"
	"mylib/internal/server"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	kApiAuthorsPath = "/api/authors"
	kSelectAuthors  = "SELECT first_name, family_name, birth_date, death_date FROM authors"
	kDeleteAuthors  = "TRUNCATE authors"
)

type Author struct {
	first_name  string
	family_name string
	birth_date  sql.NullTime
	death_date  sql.NullTime
}

func assertDateEqual(t *testing.T, time_date sql.NullTime, expected_data map[string]string, key string) {
	expected_date, ok := expected_data[key]
	assert.Equal(t, ok, time_date.Valid)
	if ok {
		assert.Equal(t, time_date.Time.Format("02.01.2006"), expected_date)
	}
}

func assertEqual(t *testing.T, author Author, expected_data map[string]string) {
	assert.Equal(t, author.first_name, expected_data["first_name"])
	assert.Equal(t, author.family_name, expected_data["family_name"])
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
		err := rows.Scan(&a.first_name, &a.family_name, &a.birth_date, &a.death_date)
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
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	request := map[string]string{
		"first_name":  "Leo",
		"family_name": "Tolstoy",
		"birth_date":  "09.09.1828",
		"death_date":  "20.11.1910",
	}
	body, _ := json.Marshal(request)

	response, err := http.Post(server.URL+kApiAuthorsPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
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
		"first_name":  "Leo",
		"family_name": "Tolstoy",
	}
	body, _ := json.Marshal(request)

	response, err := http.Post(server.URL+kApiAuthorsPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
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
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

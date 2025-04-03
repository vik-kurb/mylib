package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mylib/internal/database"
	"mylib/internal/server"
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
	kApiAuthorsPath = "/api/authors"
	kSelectAuthors  = "SELECT first_name, family_name, birth_date, death_date FROM authors"
	kDeleteAuthors  = "TRUNCATE authors"
	kTimeFormat     = "02.01.2006"
)

type Author struct {
	id          string
	first_name  string
	family_name string
	birth_date  sql.NullTime
	death_date  sql.NullTime
}

func assertDateEqual(t *testing.T, time_date sql.NullTime, expected_data map[string]string, key string) {
	expected_date, ok := expected_data[key]
	assert.Equal(t, ok, time_date.Valid)
	if ok {
		assert.Equal(t, time_date.Time.Format(kTimeFormat), expected_date)
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

func AddAuthorsDB(db *sql.DB, authors []Author) {
	for _, author := range authors {
		db.Exec("INSERT INTO authors(id, first_name, family_name, birth_date, death_date) VALUES ($1, $2, $3, $4, $5)", author.id, author.first_name, author.family_name, author.birth_date, author.death_date)
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

func TestGetAuthors_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	db_authors := []Author{
		{id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", first_name: "Alexander", family_name: "Pushkin"},
		{id: "0254a622-68fd-4812-a0bc-d997bbe3a731", first_name: "Leo", family_name: "Tolstoy"},
		{id: "1a05bda1-266c-4226-a662-51cbc60ddc86", first_name: "Fyodor", family_name: "Dostoevsky"},
	}
	AddAuthorsDB(db, db_authors)

	server := setupTestServer(db)
	defer server.Close()

	response, err := http.Get(server.URL + kApiAuthorsPath)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	type responseAuthor struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	}
	response_body := make([]responseAuthor, 0)
	err = decoder.Decode(&response_body)
	assert.NoError(t, err)
	assert.Equal(t, response_body, []responseAuthor{{Name: "Fyodor Dostoevsky", Id: "1a05bda1-266c-4226-a662-51cbc60ddc86"}, {Name: "Alexander Pushkin", Id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea"}, {Name: "Leo Tolstoy", Id: "0254a622-68fd-4812-a0bc-d997bbe3a731"}})
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
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestGetAuthorsId_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	author := Author{
		id: "aeecbc4e-9547-4fce-88ac-4e739567a1ea", first_name: "Alexander", family_name: "Pushkin", birth_date: toSqlNullTime("06.06.1799"), death_date: toSqlNullTime("10.20.1837"),
	}
	AddAuthorsDB(db, []Author{author})

	server := setupTestServer(db)
	defer server.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", server.URL, kApiAuthorsPath, author.id))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	type responseAuthor struct {
		FirstName  string `json:"first_name"`
		FamilyName string `json:"family_name"`
		BirthDate  string `json:"birth_date,omitempty"`
		DeathDate  string `json:"death_date,omitempty"`
	}
	response_body := responseAuthor{}
	err = decoder.Decode(&response_body)
	assert.NoError(t, err)
	assert.Equal(t, response_body, responseAuthor{FirstName: author.first_name, FamilyName: author.family_name, BirthDate: author.birth_date.Time.Format(kTimeFormat), DeathDate: author.death_date.Time.Format(kTimeFormat)})
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
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

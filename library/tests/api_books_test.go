package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/bakurvik/mylib/common"
	"github.com/bakurvik/mylib/library/internal/server"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	selectBooks       = "SELECT id, title, created_at, updated_at FROM books"
	insertBook        = "INSERT INTO books(id, title) VALUES ($1, $2)"
	selectBookAuthors = "SELECT author_id FROM book_authors WHERE book_id = $1 ORDER BY author_id"
	insertBookAuthors = "INSERT INTO book_authors(book_id, author_id) SELECT * FROM UNNEST($1::UUID[], $2::UUID[])"
)

type Book struct {
	id        string
	title     string
	createdAt time.Time
	updatedAt time.Time
}

func AddBooksDB(db *sql.DB, books []Book) {
	for _, book := range books {
		_, err := db.Exec(
			insertBook,
			book.id, book.title, book.createdAt, book.updatedAt)
		if err != nil {
			log.Print("Failed to add book to db: ", err)
		}
	}
}

func AddBookAuthorsDB(db *sql.DB, books []uuid.UUID, authors []uuid.UUID) {
	_, err := db.Exec(
		insertBookAuthors, books, authors)
	if err != nil {
		log.Print("Failed to add book author to db: ", err)
	}
}

func GetDbBooks(t *testing.T, db *sql.DB) []Book {
	rows, err := db.Query(selectBooks)
	if err != nil {
		t.Fatalf("Error while selecting books: %v", err)
	}
	defer common.CloseRows(rows)
	books := make([]Book, 0)

	for rows.Next() {
		b := Book{}
		err := rows.Scan(&b.id, &b.title, &b.createdAt, &b.updatedAt)
		if err != nil {
			log.Fatal("Error scanning row:", err)
		}
		books = append(books, b)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error reading rows:", err)
	}
	return books
}

func GetDbBookAuthors(t *testing.T, db *sql.DB, book_id string) []uuid.UUID {
	rows, err := db.Query(selectBookAuthors, book_id)
	if err != nil {
		t.Fatalf("Error while selecting book authors: %v", err)
	}
	defer common.CloseRows(rows)
	authors := make([]uuid.UUID, 0)

	for rows.Next() {
		authorID := uuid.UUID{}
		err := rows.Scan(&authorID)
		if err != nil {
			log.Fatal("Error scanning row:", err)
		}
		authors = append(authors, authorID)
	}

	if err := rows.Err(); err != nil {
		log.Fatal("Error reading rows:", err)
	}
	return authors
}

func TestCreateBook_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	author := Author{id: "0254a622-68fd-4812-a0bc-d997bbe3a731", fullName: "Leo Tolstoy"}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	requestBook := server.RequestBook{Title: "War and Peace", Authors: []string{author.id}}
	body, _ := json.Marshal(requestBook)

	response, err := http.Post(s.URL+server.ApiBooksPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	books := GetDbBooks(t, db)
	assert.Equal(t, len(books), 1)
	assert.Equal(t, books[0].title, requestBook.Title)

	authors := GetDbBookAuthors(t, db, books[0].id)
	assert.Equal(t, len(authors), 1)
	assert.Equal(t, authors[0].String(), author.id)
}

func TestCreateBook_SeveralAuthors(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	author1 := Author{id: "0254a622-68fd-4812-a0bc-d997bbe3a731", fullName: "Ilya Ilf"}
	author2 := Author{id: "0254a622-68fd-4812-a0bc-d997bbe3a734", fullName: "Yevgeny Petrov"}
	AddAuthorsDB(db, []Author{author1, author2})

	s := setupTestServer(db)
	defer s.Close()

	unknownAuthorId := "0254a622-68fd-4812-a0bc-d997bbe3a740"
	requestBook := server.RequestBook{Title: "The Twelve Chairs", Authors: []string{author1.id, author2.id, unknownAuthorId}}
	body, _ := json.Marshal(requestBook)

	response, err := http.Post(s.URL+server.ApiBooksPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	books := GetDbBooks(t, db)
	assert.Equal(t, len(books), 1)
	assert.Equal(t, books[0].title, requestBook.Title)

	authors := GetDbBookAuthors(t, db, books[0].id)
	assert.Equal(t, len(authors), 2)
	assert.Equal(t, authors[0].String(), author1.id)
	assert.Equal(t, authors[1].String(), author2.id)
}

func TestCreateBook_NoAuthors(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	requestBook := server.RequestBook{Title: "The Twelve Chairs"}
	body, _ := json.Marshal(requestBook)

	response, err := http.Post(s.URL+server.ApiBooksPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	books := GetDbBooks(t, db)
	assert.Equal(t, len(books), 1)
	assert.Equal(t, books[0].title, requestBook.Title)

	authors := GetDbBookAuthors(t, db, books[0].id)
	assert.Equal(t, len(authors), 0)
}

func TestCreateBook_BadRequest(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	requestBook := server.RequestBook{}
	body, _ := json.Marshal(requestBook)

	response, err := http.Post(s.URL+server.ApiBooksPath, "application/json", bytes.NewBuffer(body))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

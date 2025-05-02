package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/library/internal/server"
	"github.com/google/uuid"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	selectBooks       = "SELECT id, title, created_at, updated_at FROM books ORDER BY title"
	insertBook        = "INSERT INTO books(id, title) VALUES ($1, $2)"
	selectBookAuthors = "SELECT author_id FROM book_authors ba JOIN authors a ON ba.author_id = a.id WHERE book_id = $1 ORDER BY a.full_name"
	insertBookAuthors = "INSERT INTO book_authors(book_id, author_id) SELECT $1::uuid, UNNEST($2::text[])::uuid"
)

type Book struct {
	id        uuid.UUID
	title     string
	createdAt time.Time
	updatedAt time.Time
}

func AddBooksDB(db *sql.DB, books []Book) {
	for _, book := range books {
		_, err := db.Exec(
			insertBook,
			book.id, book.title)
		if err != nil {
			log.Print("Failed to add book to db: ", err)
		}
	}
}

func AddBookAuthorsDB(db *sql.DB, bookId string, authors []string) {
	_, err := db.Exec(
		insertBookAuthors, bookId, pq.Array(authors))
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

func GetDbBookAuthors(t *testing.T, db *sql.DB, book_id uuid.UUID) []uuid.UUID {
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

	author := Author{id: uuid.New(), fullName: "Leo Tolstoy"}
	AddAuthorsDB(db, []Author{author})

	s := setupTestServer(db)
	defer s.Close()

	requestBook := server.RequestBook{Title: "War and Peace", Authors: []string{author.id.String()}}
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
	assert.Equal(t, authors[0], author.id)
}

func TestCreateBook_SeveralAuthors(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	author1 := Author{id: uuid.New(), fullName: "Ilya Ilf"}
	author2 := Author{id: uuid.New(), fullName: "Yevgeny Petrov"}
	AddAuthorsDB(db, []Author{author1, author2})

	s := setupTestServer(db)
	defer s.Close()

	unknownAuthorId := uuid.New()
	requestBook := server.RequestBook{Title: "The Twelve Chairs", Authors: []string{author1.id.String(), author2.id.String(), unknownAuthorId.String()}}
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
	assert.Equal(t, authors[0], author1.id)
	assert.Equal(t, authors[1], author2.id)
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

func TestUpdateBook_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	authors := []Author{
		{id: uuid.New(), fullName: "Alexander Pushkin"},
		{id: uuid.New(), fullName: "Leo Tolstoy"},
	}
	AddAuthorsDB(db, authors)
	book := Book{id: uuid.New(), title: "War and Peace"}
	AddBooksDB(db, []Book{book})
	AddBookAuthorsDB(db, book.id.String(), []string{authors[1].id.String()})

	s := setupTestServer(db)
	defer s.Close()

	requestBook := server.RequestBookWithID{Id: book.id.String(), Title: "The Captain's Daughter", Authors: []string{authors[0].id.String()}}
	body, _ := json.Marshal(requestBook)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiBooksPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	books := GetDbBooks(t, db)
	assert.Equal(t, len(books), 1)
	assert.Equal(t, books[0].title, requestBook.Title)

	dbAuthors := GetDbBookAuthors(t, db, books[0].id)
	assert.Equal(t, len(dbAuthors), 1)
	assert.Equal(t, dbAuthors[0].String(), requestBook.Authors[0])
}

func TestUpdateBook_MergeAuthors(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	authors := []Author{
		{id: uuid.New(), fullName: "Alexander Pushkin"},
		{id: uuid.New(), fullName: "Leo Tolstoy"},
		{id: uuid.New(), fullName: "Fyodor Dostoevsky"},
	}
	AddAuthorsDB(db, authors)
	book := Book{id: uuid.New(), title: "War and Peace"}
	AddBooksDB(db, []Book{book})
	AddBookAuthorsDB(db, book.id.String(), []string{authors[0].id.String(), authors[1].id.String()})

	s := setupTestServer(db)
	defer s.Close()

	newAuthors := []string{authors[0].id.String(), authors[2].id.String(), uuid.New().String()}
	requestBook := server.RequestBookWithID{Id: book.id.String(), Title: "The Captain's Daughter", Authors: newAuthors}
	body, _ := json.Marshal(requestBook)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiBooksPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	books := GetDbBooks(t, db)
	assert.Equal(t, len(books), 1)
	assert.Equal(t, books[0].title, requestBook.Title)

	dbAuthors := GetDbBookAuthors(t, db, books[0].id)
	assert.Equal(t, len(dbAuthors), 2)
	assert.Equal(t, dbAuthors[0].String(), requestBook.Authors[0])
	assert.Equal(t, dbAuthors[1].String(), requestBook.Authors[1])
}

func TestUpdateBook_UnknownBook(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	requestBook := server.RequestBookWithID{Id: uuid.New().String(), Title: "The Captain's Daughter", Authors: []string{}}
	body, _ := json.Marshal(requestBook)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiBooksPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestUpdateBook_BadRequest(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	requestBook := server.RequestBookWithID{Id: "invalid_id", Title: "The Captain's Daughter", Authors: []string{}}
	body, _ := json.Marshal(requestBook)

	client := &http.Client{}
	request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiBooksPath), bytes.NewBuffer(body))
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestGetBooks_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	authors := []Author{
		{id: uuid.New(), fullName: "Alexander Pushkin"},
		{id: uuid.New(), fullName: "Leo Tolstoy"},
	}
	AddAuthorsDB(db, authors)
	book := Book{id: uuid.New(), title: "War and Peace"}
	AddBooksDB(db, []Book{book})
	AddBookAuthorsDB(db, book.id.String(), []string{authors[0].id.String(), authors[1].id.String()})

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiBooksPath, book.id))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := server.ResponseBookFullInfo{}
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.Equal(t, responseBody, server.ResponseBookFullInfo{Title: book.title, Authors: []string{authors[0].fullName, authors[1].fullName}})
}

func TestGetBooks_InvalidId(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiBooksPath, "invalid_id"))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestGetBooks_UnknownId(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiBooksPath, uuid.New()))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestDeleteBook_Success(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	authors := []Author{
		{id: uuid.New(), fullName: "Alexander Pushkin"},
		{id: uuid.New(), fullName: "Leo Tolstoy"},
	}
	AddAuthorsDB(db, authors)
	book := Book{id: uuid.New(), title: "War and Peace"}
	AddBooksDB(db, []Book{book})
	AddBookAuthorsDB(db, book.id.String(), []string{authors[0].id.String(), authors[1].id.String()})

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminBooksPath, book.id), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	books := GetDbBooks(t, db)
	assert.Equal(t, len(books), 0)

	dbAuthors := GetDbBookAuthors(t, db, book.id)
	assert.Equal(t, len(dbAuthors), 0)
}

func TestDeleteBook_InvalidId(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminBooksPath, "invalid_id"), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
}

func TestDeleteBook_UnknownId(t *testing.T) {
	db, err := common.SetupDB("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	client := &http.Client{}
	request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminBooksPath, uuid.New()), nil)
	assert.NoError(t, err)

	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

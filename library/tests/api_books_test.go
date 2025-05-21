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

func AddBookAuthorsDB(db *sql.DB, bookID string, authors []string) {
	_, err := db.Exec(
		insertBookAuthors, bookID, pq.Array(authors))
	if err != nil {
		log.Print("Failed to add book author to db: ", err)
	}
}

func GetDBBooks(t *testing.T, db *sql.DB) []Book {
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

func GetDBBookAuthors(t *testing.T, db *sql.DB, book_id uuid.UUID) []uuid.UUID {
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

func TestCreateBook(t *testing.T) {
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	type testCase struct {
		name                    string
		dbAuthors               []Author
		requestBook             server.RequestBook
		expectedStatusCode      int
		expectedDBBookTitle     string
		expectedDBBookAuthorIDs []uuid.UUID
	}
	tests := []testCase{
		{
			name:                    "success",
			dbAuthors:               []Author{{id: authorID1, fullName: "Leo Tolstoy"}},
			requestBook:             server.RequestBook{Title: "War and Peace", Authors: []string{authorID1.String()}},
			expectedStatusCode:      http.StatusCreated,
			expectedDBBookTitle:     "War and Peace",
			expectedDBBookAuthorIDs: []uuid.UUID{authorID1},
		},
		{
			name:                    "several_authors",
			dbAuthors:               []Author{{id: authorID1, fullName: "Ilya Ilf"}, {id: authorID2, fullName: "Yevgeny Petrov"}},
			requestBook:             server.RequestBook{Title: "The Twelve Chairs", Authors: []string{authorID1.String(), authorID2.String(), uuid.NewString()}},
			expectedStatusCode:      http.StatusCreated,
			expectedDBBookTitle:     "The Twelve Chairs",
			expectedDBBookAuthorIDs: []uuid.UUID{authorID1, authorID2},
		},
		{
			name:                    "no_authors",
			dbAuthors:               nil,
			requestBook:             server.RequestBook{Title: "War and Peace"},
			expectedStatusCode:      http.StatusCreated,
			expectedDBBookTitle:     "War and Peace",
			expectedDBBookAuthorIDs: nil,
		},
		{
			name:                    "bad_request",
			dbAuthors:               []Author{},
			requestBook:             server.RequestBook{},
			expectedStatusCode:      http.StatusBadRequest,
			expectedDBBookTitle:     "",
			expectedDBBookAuthorIDs: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			AddAuthorsDB(db, tc.dbAuthors)

			s := setupTestServer(db)
			defer s.Close()

			body, _ := json.Marshal(tc.requestBook)
			response, err := http.Post(s.URL+server.ApiBooksPath, "application/json", bytes.NewBuffer(body))
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.expectedDBBookTitle != "" {
				books := GetDBBooks(t, db)
				assert.Equal(t, len(books), 1)
				assert.Equal(t, books[0].title, tc.expectedDBBookTitle)
				authors := GetDBBookAuthors(t, db, books[0].id)
				assert.ElementsMatch(t, authors, tc.expectedDBBookAuthorIDs)
			}
		})
	}

}

func TestUpdateBook(t *testing.T) {
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	authorID3 := uuid.New()
	bookID1 := uuid.New()
	type testCase struct {
		name                    string
		dbAuthors               []Author
		dbBooks                 []Book
		requestBook             server.RequestBookWithID
		expectedStatusCode      int
		expectedDBBookTitle     string
		expectedDBBookAuthorIDs []uuid.UUID
	}
	tests := []testCase{
		{
			name:                    "success",
			dbAuthors:               []Author{{id: authorID1, fullName: "Leo Tolstoy"}, {id: authorID2, fullName: "Alexander Pushkin"}},
			dbBooks:                 []Book{{id: bookID1, title: "War and Peace"}},
			requestBook:             server.RequestBookWithID{ID: bookID1.String(), Title: "The Captain's Daughter", Authors: []string{authorID2.String()}},
			expectedStatusCode:      http.StatusOK,
			expectedDBBookTitle:     "The Captain's Daughter",
			expectedDBBookAuthorIDs: []uuid.UUID{authorID2},
		},
		{
			name:                    "merge_authors",
			dbAuthors:               []Author{{id: authorID1, fullName: "Leo Tolstoy"}, {id: authorID2, fullName: "Alexander Pushkin"}, {id: authorID3, fullName: "Fyodor Dostoevsky"}},
			dbBooks:                 []Book{{id: bookID1, title: "War and Peace"}},
			requestBook:             server.RequestBookWithID{ID: bookID1.String(), Title: "The Captain's Daughter", Authors: []string{authorID2.String(), authorID3.String(), uuid.NewString()}},
			expectedStatusCode:      http.StatusOK,
			expectedDBBookTitle:     "The Captain's Daughter",
			expectedDBBookAuthorIDs: []uuid.UUID{authorID2, authorID3},
		},
		{
			name:                    "unknown_book",
			dbAuthors:               []Author{{id: authorID1, fullName: "Leo Tolstoy"}},
			dbBooks:                 []Book{{id: bookID1, title: "War and Peace"}},
			requestBook:             server.RequestBookWithID{ID: uuid.NewString(), Title: "The Captain's Daughter", Authors: []string{authorID1.String()}},
			expectedStatusCode:      http.StatusNotFound,
			expectedDBBookTitle:     "",
			expectedDBBookAuthorIDs: nil,
		},
		{
			name:                    "bad_request",
			dbAuthors:               []Author{{id: authorID1, fullName: "Leo Tolstoy"}},
			dbBooks:                 []Book{{id: bookID1, title: "War and Peace"}},
			requestBook:             server.RequestBookWithID{ID: "invalid_id", Title: "The Captain's Daughter", Authors: []string{authorID2.String()}},
			expectedStatusCode:      http.StatusBadRequest,
			expectedDBBookTitle:     "",
			expectedDBBookAuthorIDs: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			AddAuthorsDB(db, tc.dbAuthors)
			AddBooksDB(db, tc.dbBooks)

			s := setupTestServer(db)
			defer s.Close()

			client := &http.Client{}
			body, _ := json.Marshal(tc.requestBook)
			request, err := http.NewRequest("PUT", fmt.Sprintf("%v%v", s.URL, server.ApiBooksPath), bytes.NewBuffer(body))
			assert.NoError(t, err)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.expectedDBBookTitle != "" {
				books := GetDBBooks(t, db)
				assert.Equal(t, len(books), 1)
				assert.Equal(t, books[0].title, tc.expectedDBBookTitle)
				authors := GetDBBookAuthors(t, db, books[0].id)
				assert.ElementsMatch(t, authors, tc.expectedDBBookAuthorIDs)
			}
		})
	}

}

func TestDeleteBook(t *testing.T) {
	bookID1 := uuid.New()
	authorID1 := uuid.New()
	type testCase struct {
		name               string
		dbAuthors          []Author
		dbBooks            []Book
		requestBook        string
		expectedStatusCode int
	}
	tests := []testCase{
		{
			name:               "success",
			dbAuthors:          []Author{{id: authorID1, fullName: "Leo Tolstoy"}},
			dbBooks:            []Book{{id: bookID1, title: "War and Peace"}},
			requestBook:        bookID1.String(),
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "bad_request",
			dbAuthors:          []Author{{id: authorID1, fullName: "Leo Tolstoy"}},
			dbBooks:            []Book{{id: bookID1, title: "War and Peace"}},
			requestBook:        "invalid_id",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "unknown_id",
			dbAuthors:          nil,
			dbBooks:            nil,
			requestBook:        uuid.NewString(),
			expectedStatusCode: http.StatusNoContent,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			AddAuthorsDB(db, tc.dbAuthors)
			AddBooksDB(db, tc.dbBooks)

			s := setupTestServer(db)
			defer s.Close()

			client := &http.Client{}
			request, err := http.NewRequest("DELETE", fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminBooksPath, tc.requestBook), nil)
			assert.NoError(t, err)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.expectedStatusCode == http.StatusNoContent {
				books := GetDBBooks(t, db)
				assert.Equal(t, len(books), 0)
			}
		})
	}

}

func TestGetBooks(t *testing.T) {
	book1 := uuid.New()
	book2 := uuid.New()
	book3 := uuid.New()
	author1 := uuid.New()
	author2 := uuid.New()

	type testCase struct {
		name               string
		requestedBooks     []string
		expectedStatusCode int
		expectedResponse   []server.ResponseBookFullInfo
	}

	tests := []testCase{
		{
			name:               "success",
			requestedBooks:     []string{book1.String(), book2.String(), book3.String()},
			expectedStatusCode: http.StatusOK,
			expectedResponse: []server.ResponseBookFullInfo{
				{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}},
				{ID: book2.String(), Title: "Title 2", Authors: []string{"Author 1", "Author 2"}},
				{ID: book3.String(), Title: "Title 3", Authors: nil}},
		},
		{
			name:               "empty request",
			requestedBooks:     []string{},
			expectedStatusCode: http.StatusBadRequest,
			expectedResponse:   nil,
		},
		{
			name:               "skip invalid bookID",
			requestedBooks:     []string{book1.String(), book2.String(), book3.String(), "invalid_book_id"},
			expectedStatusCode: http.StatusOK,
			expectedResponse: []server.ResponseBookFullInfo{
				{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}},
				{ID: book2.String(), Title: "Title 2", Authors: []string{"Author 1", "Author 2"}},
				{ID: book3.String(), Title: "Title 3", Authors: nil}},
		},
		{
			name:               "skip unknown bookID",
			requestedBooks:     []string{book1.String(), book2.String(), book3.String(), uuid.NewString()},
			expectedStatusCode: http.StatusOK,
			expectedResponse: []server.ResponseBookFullInfo{
				{ID: book1.String(), Title: "Title 1", Authors: []string{"Author 1"}},
				{ID: book2.String(), Title: "Title 2", Authors: []string{"Author 1", "Author 2"}},
				{ID: book3.String(), Title: "Title 3", Authors: nil}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			AddBooksDB(db, []Book{{id: book1, title: "Title 1"}, {id: book2, title: "Title 2"}, {id: book3, title: "Title 3"}})
			AddAuthorsDB(db, []Author{{id: author1, fullName: "Author 1"}, {id: author2, fullName: "Author 2"}})
			AddBookAuthorsDB(db, book1.String(), []string{author1.String()})
			AddBookAuthorsDB(db, book2.String(), []string{author1.String(), author2.String()})

			s := setupTestServer(db)
			defer s.Close()

			requestBooks := server.RequestBookIDs{BookIDs: tc.requestedBooks}
			body, _ := json.Marshal(requestBooks)
			client := &http.Client{}
			request, err := http.NewRequest("POST", s.URL+server.ApiBooksSearchPath, bytes.NewBuffer(body))
			assert.NoError(t, err)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.expectedResponse != nil {
				decoder := json.NewDecoder(response.Body)
				responseBody := []server.ResponseBookFullInfo{}
				err = decoder.Decode(&responseBody)
				assert.NoError(t, err)

				assert.ElementsMatch(t, responseBody, tc.expectedResponse)
			}
		})
	}
}

package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/library/internal/server"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	selectAuthors = "SELECT id, full_name, birth_date, death_date, created_at, updated_at FROM authors"
	insertAuthor  = "INSERT INTO authors(id, full_name, birth_date, death_date, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)"
)

type author struct {
	id        uuid.UUID
	fullName  string
	birthDate sql.NullTime
	deathDate sql.NullTime
	createdAt time.Time
	updatedAt time.Time
}

type expectedAuthor struct {
	fullName  string
	birthDate string
	deathDate string
}

type kafkaMessage struct {
	key   []byte
	value []byte
}

type kafkaMockWriter struct {
	messages []kafkaMessage
}

func (w *kafkaMockWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	for _, msg := range msgs {
		w.messages = append(w.messages, kafkaMessage{key: msg.Key, value: msg.Value})
	}
	return nil
}

func assertAuthorKafkaMessage(t *testing.T, w server.KafkaWriter, id string, author expectedAuthor) {
	writer := w.(*kafkaMockWriter)
	assert.Equal(t, len(writer.messages), 1)
	msg := writer.messages[0]
	assert.Equal(t, msg.key, []byte(id))
	expectedMSG, err := json.Marshal(common.AuthorMessage{ID: id, FullName: author.fullName, Action: "created"})
	assert.NoError(t, err)
	assert.Equal(t, msg.value, expectedMSG)
}

func assertDateEqual(t *testing.T, timeDate sql.NullTime, expectedDate string) {
	if expectedDate == "" {
		assert.False(t, timeDate.Valid)
		return
	}
	assert.True(t, timeDate.Valid)
	assert.Equal(t, timeDate.Time.Format(common.DateFormat), expectedDate)
}

func assertEqual(t *testing.T, dbAuthors []author, expectedAuthors []expectedAuthor) {
	assert.Equal(t, len(dbAuthors), len(expectedAuthors))
	sort.Slice(dbAuthors, func(i, j int) bool { return dbAuthors[i].fullName < dbAuthors[j].fullName })
	sort.Slice(expectedAuthors, func(i, j int) bool { return expectedAuthors[i].fullName < expectedAuthors[j].fullName })
	for i, dbAuthor := range dbAuthors {
		assert.Equal(t, dbAuthor.fullName, expectedAuthors[i].fullName)
		assertDateEqual(t, dbAuthor.birthDate, expectedAuthors[i].birthDate)
		assertDateEqual(t, dbAuthor.deathDate, expectedAuthors[i].deathDate)
	}
}

func AddAuthorsDB(db *sql.DB, authors []author) {
	for _, author := range authors {
		_, err := db.Exec(
			insertAuthor,
			author.id, author.fullName, author.birthDate, author.deathDate, author.createdAt, author.updatedAt)
		if err != nil {
			log.Print("Failed to add author to db: ", err)
		}
	}
}

func setupTestServer(db *sql.DB) (*httptest.Server, server.ApiConfig) {
	maxSearchBooksLimit, err := strconv.Atoi(os.Getenv("MAX_SEARCH_BOOKS_LIMIT"))
	if err != nil {
		log.Print("Invalid MAX_SEARCH_BOOKS_LIMIT value: ", os.Getenv("MAX_SEARCH_BOOKS_LIMIT"))
	}
	maxSearcAuthorsLimit, err := strconv.Atoi(os.Getenv("MAX_SEARCH_AUTHORS_LIMIT"))
	if err != nil {
		log.Print("Invalid MAX_SEARCH_AUTHORS_LIMIT value: ", os.Getenv("MAX_SEARCH_AUTHORS_LIMIT"))
	}

	apiCfg := server.ApiConfig{DB: db, MaxSearchBooksLimit: maxSearchBooksLimit, MaxSearchAuthorsLimit: maxSearcAuthorsLimit, AuthorsKafkaWriter: &kafkaMockWriter{}}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm), apiCfg
}

func GetDBAuthors(t *testing.T, db *sql.DB) []author {
	rows, err := db.Query(selectAuthors)
	if err != nil {
		t.Fatalf("Error while selecting authors: %v", err)
	}
	defer common.CloseRows(rows)
	authors := make([]author, 0)

	for rows.Next() {
		a := author{}
		err := rows.Scan(&a.id, &a.fullName, &a.birthDate, &a.deathDate, &a.createdAt, &a.updatedAt)
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

func TestCreateAuthor(t *testing.T) {
	type testCase struct {
		name                  string
		requestAuthor         server.RequestAuthor
		expectedStatusCode    int
		expectedDBAuthors     []expectedAuthor
		expectedKafkaMessages []kafka.Message
	}
	testCases := []testCase{
		{
			name:               "success",
			requestAuthor:      server.RequestAuthor{FullName: "Leo Tolstoy", BirthDate: "09.09.1828", DeathDate: "20.11.1910"},
			expectedStatusCode: http.StatusCreated,
			expectedDBAuthors:  []expectedAuthor{{fullName: "Leo Tolstoy", birthDate: "09.09.1828", deathDate: "20.11.1910"}},
		},
		{
			name:               "empty_dates",
			requestAuthor:      server.RequestAuthor{FullName: "Leo Tolstoy"},
			expectedStatusCode: http.StatusCreated,
			expectedDBAuthors:  []expectedAuthor{{fullName: "Leo Tolstoy"}},
		},
		{
			name:               "bad_request",
			requestAuthor:      server.RequestAuthor{},
			expectedStatusCode: http.StatusBadRequest,
			expectedDBAuthors:  nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)

			s, cfg := setupTestServer(db)
			defer s.Close()

			body, _ := json.Marshal(tc.requestAuthor)
			response, err := http.Post(s.URL+server.ApiAuthorsPath, "application/json", bytes.NewBuffer(body))
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			authors := GetDBAuthors(t, db)
			assertEqual(t, authors, tc.expectedDBAuthors)
			if tc.expectedDBAuthors != nil {
				assertAuthorKafkaMessage(t, cfg.AuthorsKafkaWriter, authors[0].id.String(), tc.expectedDBAuthors[0])
			}
		})
	}
}

func TestGetAuthors(t *testing.T) {
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	authorID3 := uuid.New()
	type testCase struct {
		name               string
		dbAuthors          []author
		requestAuthor      server.RequestAuthor
		expectedStatusCode int
		expectedAuthors    []server.ResponseAuthorShortInfo
	}
	testCases := []testCase{
		{
			name: "success",
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin"},
				{id: authorID2, fullName: "Leo Tolstoy"},
				{id: authorID3, fullName: "Fyodor Dostoevsky"},
			},
			requestAuthor:      server.RequestAuthor{FullName: "Leo Tolstoy", BirthDate: "09.09.1828", DeathDate: "20.11.1910"},
			expectedStatusCode: http.StatusOK,
			expectedAuthors: []server.ResponseAuthorShortInfo{
				{FullName: "Alexander Pushkin", ID: authorID1.String()},
				{FullName: "Fyodor Dostoevsky", ID: authorID3.String()},
				{FullName: "Leo Tolstoy", ID: authorID2.String()},
			},
		},
		{
			name:               "no_authors",
			dbAuthors:          nil,
			requestAuthor:      server.RequestAuthor{FullName: "Leo Tolstoy", BirthDate: "09.09.1828", DeathDate: "20.11.1910"},
			expectedStatusCode: http.StatusOK,
			expectedAuthors:    []server.ResponseAuthorShortInfo{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			AddAuthorsDB(db, tc.dbAuthors)

			s, _ := setupTestServer(db)
			defer s.Close()

			response, err := http.Get(s.URL + server.ApiAuthorsPath)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			decoder := json.NewDecoder(response.Body)
			responseBody := make([]server.ResponseAuthorShortInfo, 0)
			err = decoder.Decode(&responseBody)
			assert.NoError(t, err)
			assert.Equal(t, responseBody, tc.expectedAuthors)
		})
	}
}

func TestGetAuthorsID(t *testing.T) {
	authorID1 := uuid.New()
	type testCase struct {
		name               string
		dbAuthors          []author
		requestAuthor      string
		expectedStatusCode int
		expectedAuthor     server.ResponseAuthorFullInfo
	}
	testCases := []testCase{
		{
			name: "success",
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837")},
			},
			requestAuthor:      authorID1.String(),
			expectedStatusCode: http.StatusOK,
			expectedAuthor:     server.ResponseAuthorFullInfo{FullName: "Alexander Pushkin", BirthDate: "06.06.1799", DeathDate: "10.02.1837"},
		},
		{
			name: "success_no_dates",
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin"},
			},
			requestAuthor:      authorID1.String(),
			expectedStatusCode: http.StatusOK,
			expectedAuthor:     server.ResponseAuthorFullInfo{FullName: "Alexander Pushkin"},
		},
		{
			name:               "not_found",
			dbAuthors:          nil,
			requestAuthor:      authorID1.String(),
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name: "success",
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837")},
			},
			requestAuthor:      "invalid_id",
			expectedStatusCode: http.StatusBadRequest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			AddAuthorsDB(db, tc.dbAuthors)

			s, _ := setupTestServer(db)
			defer s.Close()

			response, err := http.Get(fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiAuthorsPath, tc.requestAuthor))
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
				decoder := json.NewDecoder(response.Body)
				responseBody := server.ResponseAuthorFullInfo{}
				err = decoder.Decode(&responseBody)
				assert.NoError(t, err)
				assert.Equal(t, responseBody, tc.expectedAuthor)
			}
		})
	}
}

func TestDeleteAuthor(t *testing.T) {
	authorID1 := uuid.New()
	type testCase struct {
		name               string
		dbAuthors          []author
		requestAuthor      string
		expectedStatusCode int
		expectedDBAuthors  []expectedAuthor
	}
	testCases := []testCase{
		{
			name: "success",
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837")},
			},
			requestAuthor:      authorID1.String(),
			expectedStatusCode: http.StatusOK,
			expectedDBAuthors:  nil,
		},
		{
			name:               "no_author_in_db",
			dbAuthors:          nil,
			requestAuthor:      authorID1.String(),
			expectedStatusCode: http.StatusOK,
			expectedDBAuthors:  nil,
		},
		{
			name: "invalid_id",
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837")},
			},
			requestAuthor:      "invalid_id",
			expectedStatusCode: http.StatusBadRequest,
			expectedDBAuthors:  []expectedAuthor{{fullName: "Alexander Pushkin", birthDate: "06.06.1799", deathDate: "10.02.1837"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			AddAuthorsDB(db, tc.dbAuthors)

			s, _ := setupTestServer(db)
			defer s.Close()

			client := &http.Client{}
			request, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("%v%v/{%v}", s.URL, server.AdminAuthorsPath, tc.requestAuthor), nil)
			assert.NoError(t, err)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			authors := GetDBAuthors(t, db)
			assertEqual(t, authors, tc.expectedDBAuthors)
		})
	}
}

func TestUpdateAuthor(t *testing.T) {
	authorID1 := uuid.New()
	type testCase struct {
		name               string
		requestAuthor      server.RequestAuthorWithID
		dbAuthors          []author
		expectedStatusCode int
		expectedDBAuthors  []expectedAuthor
	}
	testCases := []testCase{
		{
			name:          "success",
			requestAuthor: server.RequestAuthorWithID{ID: authorID1.String(), FullName: "Leo Tolstoy", BirthDate: "09.09.1828", DeathDate: "20.11.1910"},
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837")},
			},
			expectedStatusCode: http.StatusOK,
			expectedDBAuthors:  []expectedAuthor{{fullName: "Leo Tolstoy", birthDate: "09.09.1828", deathDate: "20.11.1910"}},
		},
		{
			name:          "not_found",
			requestAuthor: server.RequestAuthorWithID{ID: uuid.NewString(), FullName: "Leo Tolstoy", BirthDate: "09.09.1828", DeathDate: "20.11.1910"},
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837")},
			},
			expectedStatusCode: http.StatusNotFound,
			expectedDBAuthors:  []expectedAuthor{{fullName: "Alexander Pushkin", birthDate: "06.06.1799", deathDate: "10.02.1837"}},
		},
		{
			name:          "invalid_id",
			requestAuthor: server.RequestAuthorWithID{ID: "invalid_id", FullName: "Leo Tolstoy", BirthDate: "09.09.1828", DeathDate: "20.11.1910"},
			dbAuthors: []author{
				{id: authorID1, fullName: "Alexander Pushkin", birthDate: common.ToNullTime("06.06.1799"), deathDate: common.ToNullTime("10.02.1837")},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedDBAuthors:  []expectedAuthor{{fullName: "Alexander Pushkin", birthDate: "06.06.1799", deathDate: "10.02.1837"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			AddAuthorsDB(db, tc.dbAuthors)

			s, _ := setupTestServer(db)
			defer s.Close()

			client := &http.Client{}
			body, _ := json.Marshal(tc.requestAuthor)
			request, err := http.NewRequest(http.MethodPut, fmt.Sprintf("%v%v", s.URL, server.ApiAuthorsPath), bytes.NewBuffer(body))
			assert.NoError(t, err)

			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			authors := GetDBAuthors(t, db)
			assertEqual(t, authors, tc.expectedDBAuthors)
		})
	}
}

func TestGetAuthorBooks(t *testing.T) {
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	bookID1 := uuid.New()
	bookID2 := uuid.New()
	bookID3 := uuid.New()
	type testCase struct {
		name               string
		requestAuthor      string
		expectedStatusCode int
		expectedBooks      []server.ResponseBook
	}
	testCases := []testCase{
		{
			name:               "success",
			requestAuthor:      authorID1.String(),
			expectedStatusCode: http.StatusOK,
			expectedBooks:      []server.ResponseBook{{ID: bookID1.String(), Title: "Title 1"}, {ID: bookID2.String(), Title: "Title 2"}},
		},
		{
			name:               "unknown_author",
			requestAuthor:      uuid.NewString(),
			expectedStatusCode: http.StatusNotFound,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			AddAuthorsDB(db, []author{
				{id: authorID1, fullName: "Alexander Pushkin"},
				{id: authorID2, fullName: "Leo Tolstoy"},
			})
			AddBooksDB(db, []Book{{id: bookID1, title: "Title 1"}, {id: bookID2, title: "Title 2"}, {id: bookID3, title: "Title 3"}})
			AddBookAuthorsDB(db, bookID1.String(), []string{authorID1.String()})
			AddBookAuthorsDB(db, bookID2.String(), []string{authorID1.String()})
			AddBookAuthorsDB(db, bookID3.String(), []string{authorID2.String()})

			s, _ := setupTestServer(db)
			defer s.Close()

			response, err := http.Get(fmt.Sprintf("%v%v/{%v}/books", s.URL, server.ApiAuthorsPath, tc.requestAuthor))
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.expectedStatusCode == http.StatusOK {
				decoder := json.NewDecoder(response.Body)
				responseBody := make([]server.ResponseBook, 0)
				err = decoder.Decode(&responseBody)
				assert.NoError(t, err)
				assert.Equal(t, responseBody, tc.expectedBooks)
			}
		})
	}
}

func TestPing_Success(t *testing.T) {
	db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)

	s, _ := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.PingPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestSearchAuthors(t *testing.T) {
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	authorID3 := uuid.New()
	type testCase struct {
		name               string
		requestSearchText  string
		expectedStatusCode int
		expectedAuthors    []server.ResponseAuthorShortInfo
	}
	testCases := []testCase{
		{
			name:               "success",
			requestSearchText:  "Alexander",
			expectedStatusCode: http.StatusOK,
			expectedAuthors: []server.ResponseAuthorShortInfo{
				{FullName: "Alexander Alexander Pushkin", ID: authorID1.String()},
				{FullName: "Alexander Belyaev", ID: authorID2.String()},
			},
		},
		{
			name:               "empty search text",
			requestSearchText:  "",
			expectedStatusCode: http.StatusBadRequest,
			expectedAuthors:    nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			AddAuthorsDB(db, []author{
				{id: authorID1, fullName: "Alexander Alexander Pushkin"},
				{id: authorID2, fullName: "Alexander Belyaev"},
				{id: authorID3, fullName: "Fyodor Dostoevsky"},
			})

			s, _ := setupTestServer(db)
			defer s.Close()

			response, err := http.Get(s.URL + server.ApiAuthorsSearchPath + "?text=" + tc.requestSearchText)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.expectedAuthors != nil {
				decoder := json.NewDecoder(response.Body)
				responseBody := make([]server.ResponseAuthorShortInfo, 0)
				err = decoder.Decode(&responseBody)
				assert.NoError(t, err)
				assert.Equal(t, responseBody, tc.expectedAuthors)
			}
		})
	}
}

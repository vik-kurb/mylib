package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bakurvik/mylib/library/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMergeAuthors(t *testing.T) {
	type testCase struct {
		name                    string
		oldAuthors              []uuid.UUID
		newAuthors              []uuid.UUID
		expectedAuthorsToDelete []uuid.UUID
		expectedAuthorsToInsert []uuid.UUID
	}
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	authorID3 := uuid.New()
	authorID4 := uuid.New()
	authorID5 := uuid.New()
	testCases := []testCase{
		{
			name:                    "no_change",
			oldAuthors:              []uuid.UUID{authorID1, authorID2, authorID3},
			newAuthors:              []uuid.UUID{authorID1, authorID2, authorID3},
			expectedAuthorsToDelete: nil,
			expectedAuthorsToInsert: nil,
		},
		{
			name:                    "only_delete",
			oldAuthors:              []uuid.UUID{authorID1, authorID2, authorID3},
			newAuthors:              []uuid.UUID{authorID1},
			expectedAuthorsToDelete: []uuid.UUID{authorID2, authorID3},
			expectedAuthorsToInsert: nil,
		},
		{
			name:                    "only_insert",
			oldAuthors:              []uuid.UUID{authorID1, authorID2, authorID3},
			newAuthors:              []uuid.UUID{authorID1, authorID2, authorID3, authorID4, authorID5},
			expectedAuthorsToDelete: nil,
			expectedAuthorsToInsert: []uuid.UUID{authorID4, authorID5},
		},
		{
			name:                    "delete_and_insert",
			oldAuthors:              []uuid.UUID{authorID1, authorID2, authorID3},
			newAuthors:              []uuid.UUID{authorID1, authorID4, authorID5},
			expectedAuthorsToDelete: []uuid.UUID{authorID2, authorID3},
			expectedAuthorsToInsert: []uuid.UUID{authorID4, authorID5},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			diff := mergeAuthors(tc.oldAuthors, tc.newAuthors)

			assert.ElementsMatch(t, diff.authorsToDelete, tc.expectedAuthorsToDelete)
			assert.ElementsMatch(t, diff.authorsToInsert, tc.expectedAuthorsToInsert)
		})
	}
}

func TestGetUniqueAuthors(t *testing.T) {
	bookID1 := uuid.New()
	bookID2 := uuid.New()
	bookID3 := uuid.New()
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	authorID3 := uuid.New()
	type testCase struct {
		name              string
		bookAuthors       []database.GetAuthorsByBooksRow
		expectedAuthorIDS []uuid.UUID
	}
	testCases := []testCase{
		{
			name: "remove_duplicate",
			bookAuthors: []database.GetAuthorsByBooksRow{
				{BookID: bookID1, AuthorID: authorID1},
				{BookID: bookID2, AuthorID: authorID1},
				{BookID: bookID2, AuthorID: authorID2},
				{BookID: bookID3, AuthorID: authorID2},
				{BookID: bookID3, AuthorID: authorID3},
			},
			expectedAuthorIDS: []uuid.UUID{authorID1, authorID2, authorID3},
		},
		{
			name:              "empty",
			bookAuthors:       []database.GetAuthorsByBooksRow{},
			expectedAuthorIDS: []uuid.UUID{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			authorIDs := getUniqueAuthors(tc.bookAuthors)
			assert.ElementsMatch(t, authorIDs, tc.expectedAuthorIDS)
		})
	}
}

func makeRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/test", io.NopCloser(strings.NewReader(body)))
	return r
}

func TestParseBookIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	type testCase struct {
		name            string
		body            string
		expectedBookIDs []uuid.UUID
		hasError        bool
	}
	testCases := []testCase{
		{
			name:            "all_valid",
			body:            fmt.Sprintf(`{"book_ids": ["%v", "%v"]}`, id1, id2),
			expectedBookIDs: []uuid.UUID{id1, id2},
			hasError:        false,
		},
		{
			name:            "filter_out_invalid",
			body:            fmt.Sprintf(`{"book_ids": ["%v", "%v", "invalid_id"]}`, id1, id2),
			expectedBookIDs: []uuid.UUID{id1, id2},
			hasError:        false,
		},
		{
			name:            "empty",
			body:            `{"book_ids": []}`,
			expectedBookIDs: nil,
			hasError:        true,
		},
		{
			name:            "invalid_json",
			body:            `invalid_json`,
			expectedBookIDs: nil,
			hasError:        true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := makeRequest(tc.body)
			uuids, err := parseBookIDs(req)
			assert.Equal(t, err != nil, tc.hasError)
			assert.ElementsMatch(t, uuids, tc.expectedBookIDs)
		})
	}
}

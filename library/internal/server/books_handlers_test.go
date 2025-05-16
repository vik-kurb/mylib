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

func TestMergeAuthors_NoDiff(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := oldAuthors
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.Equal(t, len(diff.authorsToDelete), 0)
	assert.Equal(t, len(diff.authorsToInsert), 0)
}

func TestMergeAuthors_OnlyDelete(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := []uuid.UUID{oldAuthors[0]}
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.ElementsMatch(t, diff.authorsToDelete, []uuid.UUID{oldAuthors[1], oldAuthors[2]})
	assert.Equal(t, len(diff.authorsToInsert), 0)
}

func TestMergeAuthors_OnlyInsert(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := oldAuthors
	newAuthors = append(newAuthors, uuid.New(), uuid.New())
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.ElementsMatch(t, diff.authorsToInsert, []uuid.UUID{newAuthors[3], newAuthors[4]})
	assert.Equal(t, len(diff.authorsToDelete), 0)
}

func TestMergeAuthors_DeleteAndInsert(t *testing.T) {
	oldAuthors := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	newAuthors := []uuid.UUID{oldAuthors[0], uuid.New(), uuid.New()}
	diff := mergeAuthors(oldAuthors, newAuthors)

	assert.ElementsMatch(t, diff.authorsToInsert, []uuid.UUID{newAuthors[1], newAuthors[2]})
	assert.ElementsMatch(t, diff.authorsToDelete, []uuid.UUID{oldAuthors[1], oldAuthors[2]})
}

func TestGetUniqueAuthors(t *testing.T) {
	bookID1 := uuid.New()
	bookID2 := uuid.New()
	bookID3 := uuid.New()
	authorID1 := uuid.New()
	authorID2 := uuid.New()
	authorID3 := uuid.New()
	bookAuthors := []database.GetAuthorsByBooksRow{
		{BookID: bookID1, AuthorID: authorID1},
		{BookID: bookID2, AuthorID: authorID1},
		{BookID: bookID2, AuthorID: authorID2},
		{BookID: bookID3, AuthorID: authorID2},
		{BookID: bookID3, AuthorID: authorID3},
	}
	authorIDs := getUniqueAuthors(bookAuthors)
	assert.ElementsMatch(t, authorIDs, []uuid.UUID{authorID1, authorID2, authorID3})
}

func TestGetUniqueAuthors_Empty(t *testing.T) {
	bookAuthors := []database.GetAuthorsByBooksRow{}
	authorIDs := getUniqueAuthors(bookAuthors)
	assert.ElementsMatch(t, authorIDs, []uuid.UUID{})
}

func makeRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/test", io.NopCloser(strings.NewReader(body)))
	return r
}

func TestParseBookIDs_AllValid(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	body := fmt.Sprintf(`{"book_ids": ["%v", "%v"]}`, id1, id2)

	req := makeRequest(body)
	uuids, err := parseBookIds(req)
	assert.NoError(t, err)
	assert.ElementsMatch(t, uuids, []uuid.UUID{id1, id2})
}

func TestParseBookIDs_FilterOutInvalid(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	invalidID := "invalidID"
	body := fmt.Sprintf(`{"book_ids": ["%v", "%v", "%v"]}`, id1, id2, invalidID)

	req := makeRequest(body)
	uuids, err := parseBookIds(req)
	assert.NoError(t, err)
	assert.ElementsMatch(t, uuids, []uuid.UUID{id1, id2})
}

func TestParseBookIDs_Empty(t *testing.T) {
	body := `{"book_ids": []}`

	req := makeRequest(body)
	uuids, err := parseBookIds(req)
	assert.Error(t, err)
	assert.ElementsMatch(t, uuids, nil)
}

func TestParseBookIDs_InvalidJSON(t *testing.T) {
	body := `invalid_json`

	req := makeRequest(body)
	uuids, err := parseBookIds(req)
	assert.Error(t, err)
	assert.ElementsMatch(t, uuids, nil)
}

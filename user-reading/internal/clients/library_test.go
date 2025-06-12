package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/user-reading/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func mockLibraryServer(t *testing.T, statusCode int, response []ResponseBookFullInfo, expectedBookIDs []string) *url.URL {
	libraryServiceMock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == LibraryApiBooksSearchPath {
			decoder := json.NewDecoder(r.Body)
			requestData := RequestBookIDs{}
			err := decoder.Decode(&requestData)
			assert.NoError(t, err)
			assert.ElementsMatch(t, requestData.BookIDs, expectedBookIDs)
			common.RespondWithJSON(w, statusCode, response, nil)
			return
		}
		http.NotFound(w, r)
	}))

	libraryURL, _ := url.Parse(libraryServiceMock.URL)
	return libraryURL
}

func TestGetBooksInfo(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	type testCase struct {
		name                          string
		enableCache                   bool
		libraryStatusCode             int
		libraryResponse               []ResponseBookFullInfo
		bookIDs                       []string
		expectedLibraryRequestBookIDs []string
		expectedBooks                 []ResponseBookFullInfo
		hasError                      bool
	}
	testCases := []testCase{
		{
			name:              "do_not_use_cache",
			enableCache:       false,
			libraryStatusCode: http.StatusOK,
			libraryResponse: []ResponseBookFullInfo{
				{ID: id1.String(), Title: "Title 1", Authors: []string{"Author 1", "Author 2"}},
				{ID: id2.String(), Title: "Title 2", Authors: []string{"Author 3"}},
			},
			bookIDs:                       []string{id1.String(), id2.String()},
			expectedLibraryRequestBookIDs: []string{id1.String(), id2.String()},
			expectedBooks: []ResponseBookFullInfo{
				{ID: id1.String(), Title: "Title 1", Authors: []string{"Author 1", "Author 2"}},
				{ID: id2.String(), Title: "Title 2", Authors: []string{"Author 3"}},
			},
			hasError: false,
		},
		{
			name:              "cache_is_empty",
			enableCache:       true,
			libraryStatusCode: http.StatusOK,
			libraryResponse: []ResponseBookFullInfo{
				{ID: id1.String(), Title: "Title 1", Authors: []string{"Author 1", "Author 2"}},
				{ID: id2.String(), Title: "Title 2", Authors: []string{"Author 3"}},
			},
			bookIDs:                       []string{id1.String(), id2.String()},
			expectedLibraryRequestBookIDs: []string{id1.String(), id2.String()},
			expectedBooks: []ResponseBookFullInfo{
				{ID: id1.String(), Title: "Title 1", Authors: []string{"Author 1", "Author 2"}},
				{ID: id2.String(), Title: "Title 2", Authors: []string{"Author 3"}},
			},
			hasError: false,
		},
		{
			name:                          "all_data_from_cache",
			enableCache:                   true,
			libraryStatusCode:             http.StatusOK,
			libraryResponse:               []ResponseBookFullInfo{},
			bookIDs:                       []string{id1.String(), id2.String()},
			expectedLibraryRequestBookIDs: []string{},
			expectedBooks: []ResponseBookFullInfo{
				{ID: id1.String(), Title: "Title 1", Authors: []string{"Author 1", "Author 2"}},
				{ID: id2.String(), Title: "Title 2", Authors: []string{"Author 3"}},
			},
			hasError: false,
		},
		{
			name:              "part_data_from_cache",
			enableCache:       true,
			libraryStatusCode: http.StatusOK,
			libraryResponse: []ResponseBookFullInfo{
				{ID: id3.String(), Title: "Title 3", Authors: []string{"Author 4", "Author 2"}},
			},
			bookIDs:                       []string{id1.String(), id2.String(), id3.String()},
			expectedLibraryRequestBookIDs: []string{id3.String()},
			expectedBooks: []ResponseBookFullInfo{
				{ID: id1.String(), Title: "Title 1", Authors: []string{"Author 1", "Author 2"}},
				{ID: id2.String(), Title: "Title 2", Authors: []string{"Author 3"}},
				{ID: id3.String(), Title: "Title 3", Authors: []string{"Author 4", "Author 2"}},
			},
			hasError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			libraryHost := mockLibraryServer(t, tc.libraryStatusCode, tc.libraryResponse, tc.expectedLibraryRequestBookIDs)
			statusCode, books, err := GetBooksInfo(tc.bookIDs, libraryHost.String(), config.BooksCacheConfig{Enable: tc.enableCache})

			assert.Equal(t, err != nil, tc.hasError)
			assert.Equal(t, statusCode, tc.libraryStatusCode)
			assert.ElementsMatch(t, books, tc.expectedBooks)
		})
	}
}

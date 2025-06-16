package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/user-reading/internal/config"
	"github.com/google/uuid"
)

func CheckBook(bookID uuid.UUID, host string) (int, error) {
	response, err := http.Get(fmt.Sprintf("%v%v/%v", host, LibraryApiBooksPath, bookID))
	if err != nil {
		return 0, err
	}
	defer common.CloseResponseBody(response)
	return response.StatusCode, nil
}

func GetBooksInfoWithCache(bookIDs []string, host string) (int, map[string]ResponseBookFullInfo, error) {
	request := RequestBookIDs{}
	booksInfo := make(map[string]ResponseBookFullInfo)
	booksInfoMU := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, bookID := range bookIDs {
		bc.prepareLookup(bookID, &request.BookIDs, &wg, &booksInfoMU, booksInfo)
	}
	if len(request.BookIDs) == 0 {
		wg.Wait()
		return http.StatusOK, booksInfo, nil
	}

	body, _ := json.Marshal(request)
	response, err := http.Post(fmt.Sprintf("%v%v", host, LibraryApiBooksSearchPath), "application/json", bytes.NewBuffer(body))
	if err != nil {
		bc.closeChannels(request.BookIDs)
		return 0, nil, err
	}
	defer common.CloseResponseBody(response)
	decoder := json.NewDecoder(response.Body)
	responseData := []ResponseBookFullInfo{}
	err = decoder.Decode(&responseData)
	if err != nil {
		bc.closeChannels(request.BookIDs)
		return 0, nil, err
	}
	for _, bookInfo := range responseData {
		bc.update(bookInfo)
	}
	bc.closeChannels(request.BookIDs)

	wg.Wait()
	return response.StatusCode, booksInfo, nil
}

func GetBooksInfo(bookIDs []string, host string, cfg config.BooksCacheConfig) (int, map[string]ResponseBookFullInfo, error) {
	if cfg.Enable {
		return GetBooksInfoWithCache(bookIDs, host)
	}
	requestBook := RequestBookIDs{BookIDs: bookIDs}
	body, _ := json.Marshal(requestBook)
	response, err := http.Post(fmt.Sprintf("%v%v", host, LibraryApiBooksSearchPath), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return 0, nil, err
	}
	defer common.CloseResponseBody(response)
	decoder := json.NewDecoder(response.Body)
	responseData := []ResponseBookFullInfo{}
	err = decoder.Decode(&responseData)
	if err != nil {
		return 0, nil, err
	}
	result := make(map[string]ResponseBookFullInfo)
	for _, bookInfo := range responseData {
		result[bookInfo.ID] = bookInfo
	}
	return response.StatusCode, result, nil
}

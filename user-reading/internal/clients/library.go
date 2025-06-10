package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	common "github.com/bakurvik/mylib-common"
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

type cacheBookInfo struct {
	info  ResponseBookFullInfo
	ready chan struct{}
}

type booksCache struct {
	IDToInfo map[string]*cacheBookInfo
	mu       sync.Mutex
}

var bc = booksCache{IDToInfo: make(map[string]*cacheBookInfo)}

func (bc *booksCache) closeChannels(bookIDs []string) {
	for _, bookID := range bookIDs {
		bc.mu.Lock()
		cacheBook, ok := bc.IDToInfo[bookID]
		if ok {
			select {
			case <-cacheBook.ready:
			default:
				close(cacheBook.ready)
			}
			if cacheBook.info.ID == "" {
				delete(bc.IDToInfo, bookID)
			}
		}
		bc.mu.Unlock()
	}
}

func GetBooksInfoWithCache(bookIDs []string, host string) (int, []ResponseBookFullInfo, error) {
	request := RequestBookIDs{}
	booksInfo := []ResponseBookFullInfo{}
	booksInfoMU := sync.Mutex{}
	wg := sync.WaitGroup{}
	for _, bookID := range bookIDs {
		bc.mu.Lock()
		info, ok := bc.IDToInfo[bookID]
		if !ok {
			request.BookIDs = append(request.BookIDs, bookID)
			cacheBook := cacheBookInfo{ready: make(chan struct{})}
			bc.IDToInfo[bookID] = &cacheBook
			info = &cacheBook
		}
		bc.mu.Unlock()
		wg.Add(1)
		go func(info *cacheBookInfo) {
			defer wg.Done()
			<-info.ready
			booksInfoMU.Lock()
			if info.info.ID != "" {
				booksInfo = append(booksInfo, info.info)
			}
			booksInfoMU.Unlock()
		}(info)
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
		bc.mu.Lock()
		if _, ok := bc.IDToInfo[bookInfo.ID]; ok {
			bc.IDToInfo[bookInfo.ID].info = bookInfo
		}
		bc.mu.Unlock()
	}
	bc.closeChannels(request.BookIDs)

	wg.Wait()
	return response.StatusCode, booksInfo, nil
}

func GetBooksInfo(bookIDs []string, host string, useCache bool) (int, []ResponseBookFullInfo, error) {
	if useCache {
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
	return response.StatusCode, responseData, nil
}

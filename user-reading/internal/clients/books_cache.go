package clients

import (
	"log"
	"sync"
	"time"
)

type cacheBookInfo struct {
	info      ResponseBookFullInfo
	ready     chan struct{}
	updatedAt time.Time
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

func (bc *booksCache) update(bookInfo ResponseBookFullInfo) {
	bc.mu.Lock()
	if _, ok := bc.IDToInfo[bookInfo.ID]; ok {
		bc.IDToInfo[bookInfo.ID].info = bookInfo
		bc.IDToInfo[bookInfo.ID].updatedAt = time.Now()
	}
	bc.mu.Unlock()
}

func (bc *booksCache) prepareLookup(bookID string, requestBookIDs *[]string, updateInfoWG *sync.WaitGroup, booksInfoMU *sync.Mutex, booksInfo *[]ResponseBookFullInfo) {
	bc.mu.Lock()
	info, ok := bc.IDToInfo[bookID]
	if !ok {
		*requestBookIDs = append(*requestBookIDs, bookID)
		cacheBook := cacheBookInfo{ready: make(chan struct{})}
		bc.IDToInfo[bookID] = &cacheBook
		info = &cacheBook
	}
	bc.mu.Unlock()
	updateInfoWG.Add(1)
	go func(info *cacheBookInfo) {
		defer updateInfoWG.Done()
		<-info.ready
		booksInfoMU.Lock()
		if info.info.ID != "" {
			*booksInfo = append(*booksInfo, info.info)
		}
		booksInfoMU.Unlock()
	}(info)
}

type Ticker interface {
	Stop()
	C() <-chan time.Time
}

type TimeTicker struct {
	T *time.Ticker
}

func (tt *TimeTicker) C() <-chan time.Time {
	return tt.T.C
}
func (tt *TimeTicker) Stop() {
	tt.T.Stop()
}

func CleanupBooksCache(oldDataThreshold time.Duration, ticker Ticker) {
	defer ticker.Stop()
	for {
		_, ok := <-ticker.C()
		if !ok {
			return
		}
		keysToDelete := make([]string, 0)
		bc.mu.Lock()
		for key, info := range bc.IDToInfo {
			if time.Since(info.updatedAt) > oldDataThreshold {
				keysToDelete = append(keysToDelete, key)
			}
		}
		for _, key := range keysToDelete {
			delete(bc.IDToInfo, key)
		}
		bc.mu.Unlock()
		log.Printf("Deleted %v elements from books cache", len(keysToDelete))
	}
}

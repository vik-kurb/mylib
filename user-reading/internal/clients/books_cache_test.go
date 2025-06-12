package clients

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type FakeTicker struct {
	Ch chan time.Time
}

func (ft *FakeTicker) C() <-chan time.Time {
	return ft.Ch
}
func (ft *FakeTicker) Stop() {
}

func cleanupAndFillCache(books []cacheBookInfo) {
	bc.mu.Lock()
	bc.IDToInfo = make(map[string]*cacheBookInfo)
	for _, book := range books {
		bc.IDToInfo[book.info.ID] = &book
	}
	bc.mu.Unlock()
}

func TestCleanupBooksCache(t *testing.T) {
	newCacheBook := cacheBookInfo{
		info: ResponseBookFullInfo{ID: uuid.NewString()}, updatedAt: time.Now().Add(-time.Minute * 5),
	}
	oldCacheBook := cacheBookInfo{
		info: ResponseBookFullInfo{ID: uuid.NewString()}, updatedAt: time.Now().Add(-time.Hour),
	}
	veryOldCacheBook := cacheBookInfo{
		info: ResponseBookFullInfo{ID: uuid.NewString()}, updatedAt: time.Now().Add(-time.Hour * 10),
	}
	type testCase struct {
		name               string
		oldDataThreshold   time.Duration
		cacheBooks         []cacheBookInfo
		expectedCacheBooks []cacheBookInfo
	}
	testCases := []testCase{
		{
			name:               "no_deletes",
			oldDataThreshold:   time.Hour * 24,
			cacheBooks:         []cacheBookInfo{newCacheBook, oldCacheBook, veryOldCacheBook},
			expectedCacheBooks: []cacheBookInfo{newCacheBook, oldCacheBook, veryOldCacheBook},
		},
		{
			name:               "delete_some_data",
			oldDataThreshold:   time.Hour * 5,
			cacheBooks:         []cacheBookInfo{newCacheBook, oldCacheBook, veryOldCacheBook},
			expectedCacheBooks: []cacheBookInfo{newCacheBook, oldCacheBook},
		},
		{
			name:               "delete_all_data",
			oldDataThreshold:   time.Minute,
			cacheBooks:         []cacheBookInfo{newCacheBook, oldCacheBook, veryOldCacheBook},
			expectedCacheBooks: []cacheBookInfo{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cleanupAndFillCache(tc.cacheBooks)
			ft := FakeTicker{Ch: make(chan time.Time)}
			done := make(chan struct{})
			go func() {
				CleanupBooksCache(tc.oldDataThreshold, &ft)
				close(done)
			}()
			ft.Ch <- time.Now()
			close(ft.Ch)

			<-done
			cacheBooks := make([]cacheBookInfo, 0)
			bc.mu.Lock()
			for _, book := range bc.IDToInfo {
				cacheBooks = append(cacheBooks, *book)
			}
			bc.mu.Unlock()
			assert.ElementsMatch(t, cacheBooks, tc.expectedCacheBooks)
		})
	}
}

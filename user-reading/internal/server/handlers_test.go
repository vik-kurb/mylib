package server

import (
	"database/sql"
	"testing"
	"time"

	"github.com/bakurvik/mylib/user-reading/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func getOrder(books []dbUserReading) []uuid.UUID {
	res := make([]uuid.UUID, 0, len(books))
	for _, book := range books {
		res = append(res, book.bookID)
	}
	return res
}

func TestSortUserReading(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	id4 := uuid.New()
	type testCase struct {
		name               string
		books              []dbUserReading
		expectedBooksOrder []uuid.UUID
	}
	testCases := []testCase{
		{
			name: "reading_status",
			books: []dbUserReading{
				{
					bookID:     id1,
					status:     database.ReadingStatus("reading"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 50)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 1)},
					createdAt:  time.Now(),
				},
				{
					bookID:     id2,
					status:     database.ReadingStatus("reading"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 10)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 5)},
					createdAt:  time.Now(),
				},
				{
					bookID:     id3,
					status:     database.ReadingStatus("reading"),
					startDate:  sql.NullTime{Valid: false},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 100)},
					createdAt:  time.Now().Add(-time.Hour * 10),
				},
				{
					bookID:     id4,
					status:     database.ReadingStatus("reading"),
					startDate:  sql.NullTime{Valid: false},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 100)},
					createdAt:  time.Now(),
				},
			},
			expectedBooksOrder: []uuid.UUID{id4, id3, id2, id1},
		},
		{
			name: "finished_status",
			books: []dbUserReading{
				{
					bookID:     id1,
					status:     database.ReadingStatus("finished"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 50)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 5)},
					createdAt:  time.Now(),
				},
				{
					bookID:     id2,
					status:     database.ReadingStatus("finished"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 10)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 1)},
					createdAt:  time.Now(),
				},
				{
					bookID:     id3,
					status:     database.ReadingStatus("finished"),
					startDate:  sql.NullTime{Valid: false},
					finishDate: sql.NullTime{Valid: false},
					createdAt:  time.Now().Add(-time.Hour * 10),
				},
				{
					bookID:     id4,
					status:     database.ReadingStatus("finished"),
					startDate:  sql.NullTime{Valid: false},
					finishDate: sql.NullTime{Valid: false},
					createdAt:  time.Now(),
				},
			},
			expectedBooksOrder: []uuid.UUID{id4, id3, id2, id1},
		},
		{
			name: "want_to_read",
			books: []dbUserReading{
				{
					bookID:     id1,
					status:     database.ReadingStatus("want_to_read"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 50)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 5)},
					createdAt:  time.Now().Add(-time.Hour * 5),
				},
				{
					bookID:     id2,
					status:     database.ReadingStatus("want_to_read"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 10)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 1)},
					createdAt:  time.Now().Add(-time.Hour),
				},
				{
					bookID:     id3,
					status:     database.ReadingStatus("want_to_read"),
					startDate:  sql.NullTime{Valid: false},
					finishDate: sql.NullTime{Valid: false},
					createdAt:  time.Now().Add(-time.Hour * 10),
				},
				{
					bookID:     id4,
					status:     database.ReadingStatus("want_to_read"),
					startDate:  sql.NullTime{Valid: false},
					finishDate: sql.NullTime{Valid: false},
					createdAt:  time.Now(),
				},
			},
			expectedBooksOrder: []uuid.UUID{id4, id2, id1, id3},
		},
		{
			name: "mixed_reading_statuses",
			books: []dbUserReading{
				{
					bookID:     id1,
					status:     database.ReadingStatus("finished"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 50)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 1)},
					createdAt:  time.Now(),
				},
				{
					bookID:     id2,
					status:     database.ReadingStatus("reading"),
					startDate:  sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 10)},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 5)},
					createdAt:  time.Now(),
				},
				{
					bookID:     id3,
					status:     database.ReadingStatus("want_to_read"),
					startDate:  sql.NullTime{Valid: false},
					finishDate: sql.NullTime{Valid: true, Time: time.Now().Add(-time.Hour * 100)},
					createdAt:  time.Now().Add(-time.Hour * 10),
				},
			},
			expectedBooksOrder: []uuid.UUID{id2, id3, id1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sortUserReading(tc.books)
			booksOrder := getOrder(tc.books)
			assert.Equal(t, booksOrder, tc.expectedBooksOrder)
		})
	}
}

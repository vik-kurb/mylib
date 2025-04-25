package tests

import (
	"database/sql"
	"log"
)

const (
	deleteAuthors = "DELETE FROM authors"
	deleteBooks   = "DELETE FROM books"
)

func cleanupDB(db *sql.DB) {
	_, err := db.Query(deleteAuthors)
	if err != nil {
		log.Print("Failed to cleanup authors: ", err)
	}
	_, err = db.Query(deleteBooks)
	if err != nil {
		log.Print("Failed to cleanup books: ", err)
	}
}

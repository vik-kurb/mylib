package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"

	"github.com/bakurvik/mylib/common"
	"github.com/bakurvik/mylib/library/internal/database"
)

func insertBookAuthors(cfg *ApiConfig, w http.ResponseWriter, r *http.Request, authorIds []string, bookID uuid.UUID) error {
	books := make([]uuid.UUID, 0)
	authors := make([]uuid.UUID, 0)
	for _, authorID := range authorIds {
		authorUUID, err := uuid.Parse(authorID)
		if err != nil {
			log.Print("Invalid author ", authorID)
			continue
		}
		books = append(books, bookID)
		authors = append(authors, authorUUID)
	}

	tx, err := cfg.DB.BeginTx(r.Context(), nil)
	if err != nil {
		return err
	}
	defer func() {
		if err == nil {
			err = tx.Commit()
			if err != nil {
				log.Print("Failed to add book authors ", err)
			}
			return
		}
		log.Print("Failed to add book authors ", err)
		err = tx.Rollback()
		if err != nil {
			log.Print("Failed to rollback transaction ", err)
		}
	}()

	queries := database.New(tx)

	filteredAuthorIDs, err := queries.CheckAuthors(r.Context(), authors)
	if err != nil {
		return err
	}
	books = books[:len(filteredAuthorIDs)]

	dbErr := queries.AddBookAuthors(r.Context(), database.AddBookAuthorsParams{Books: books, Authors: filteredAuthorIDs})
	if dbErr != nil {
		log.Print("Failed to add book author: ", dbErr)
	}
	return nil
}

func (cfg *ApiConfig) HandlePostApiBooks(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := RequestBook{}
	err := decoder.Decode(&request)
	if err != nil || request.Title == "" {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	queries := database.New(cfg.DB)
	bookID, dbErr := queries.CreateBook(
		r.Context(), request.Title)
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}

	if len(request.Authors) > 0 {
		err = insertBookAuthors(cfg, w, r, request.Authors, bookID)
		if err != nil {
			common.RespondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	w.WriteHeader(http.StatusCreated)
}

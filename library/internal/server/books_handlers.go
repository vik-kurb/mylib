package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/google/uuid"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/library/internal/database"
)

type AuthorsDiff struct {
	authorsToInsert []uuid.UUID
	authorsToDelete []uuid.UUID
}

func insertBookAuthors(queries *database.Queries, w http.ResponseWriter, r *http.Request, authorIds []string, bookID uuid.UUID) error {
	authors := make([]uuid.UUID, 0)
	for _, authorID := range authorIds {
		authorUUID, err := uuid.Parse(authorID)
		if err != nil {
			log.Print("Invalid author ", authorID)
			continue
		}
		authors = append(authors, authorUUID)
	}

	filteredAuthorIDs, err := queries.CheckAuthors(r.Context(), authors)
	if err != nil {
		return err
	}

	dbErr := queries.AddBookAuthors(r.Context(), database.AddBookAuthorsParams{Book: bookID, Authors: filteredAuthorIDs})
	if dbErr != nil {
		return dbErr
	}
	return nil
}

func mergeAuthors(oldAuthors []uuid.UUID, newAuthors []uuid.UUID) AuthorsDiff {
	diff := AuthorsDiff{}
	diff.authorsToInsert = make([]uuid.UUID, 0)
	diff.authorsToDelete = make([]uuid.UUID, 0)

	containsFunc := func(authors []uuid.UUID, authorID uuid.UUID) bool {
		for _, a := range authors {
			if a == authorID {
				return true
			}
		}
		return false
	}

	for _, oldAuthor := range oldAuthors {
		if !containsFunc(newAuthors, oldAuthor) {
			diff.authorsToDelete = append(diff.authorsToDelete, oldAuthor)
		}
	}
	for _, newAuthor := range newAuthors {
		if !containsFunc(oldAuthors, newAuthor) {
			diff.authorsToInsert = append(diff.authorsToInsert, newAuthor)
		}
	}
	return diff
}

func updateBookAuthors(queries *database.Queries, w http.ResponseWriter, r *http.Request, authorIds []string, bookID uuid.UUID) error {
	oldAuthorIDs, dbErr := queries.GetAuthorsByBook(r.Context(), bookID)
	if dbErr != nil {
		return dbErr
	}

	newAuthors := make([]uuid.UUID, 0)
	for _, authorID := range authorIds {
		authorUUID, err := uuid.Parse(authorID)
		if err != nil {
			log.Print("Invalid author ", authorID)
			continue
		}
		newAuthors = append(newAuthors, authorUUID)
	}

	filteredNewAuthorIDs, err := queries.CheckAuthors(r.Context(), newAuthors)
	if err != nil {
		return err
	}

	authorsDiff := mergeAuthors(oldAuthorIDs, filteredNewAuthorIDs)
	if len(authorsDiff.authorsToInsert) > 0 {
		dbErr := queries.AddBookAuthors(r.Context(), database.AddBookAuthorsParams{Book: bookID, Authors: authorsDiff.authorsToInsert})
		if dbErr != nil {
			return dbErr
		}
	}
	if len(authorsDiff.authorsToDelete) > 0 {
		dbErr := queries.DeleteBookAuthors(r.Context(), database.DeleteBookAuthorsParams{BookID: bookID, Authors: authorsDiff.authorsToDelete})
		if dbErr != nil {
			return dbErr
		}
	}
	return nil
}

// @Summary Creates new book
// @Description Creates new book and stores it in DB
// @Tags Books
// @Accept json
// @Produce json
// @Param request body RequestBook true "Book's info"
// @Success 201 {string} string "Created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or empty title"
// @Failure 500 {object} ErrorResponse
// @Router /api/books [post]
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

	tx, err := cfg.DB.BeginTx(r.Context(), nil)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Print("Failed to rollback transaction ", rollbackErr)
			}
			common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		} else {
			err = tx.Commit()
			if err != nil {
				common.RespondWithError(w, http.StatusInternalServerError, err.Error())
			}
			w.WriteHeader(http.StatusCreated)
		}
	}()

	queries := database.New(tx)

	bookID, err := queries.CreateBook(r.Context(), request.Title)
	if err != nil {
		return
	}

	if len(request.Authors) > 0 {
		err = insertBookAuthors(queries, w, r, request.Authors, bookID)
		if err != nil {
			return
		}
	}
}

// @Summary Updates book
// @Description Updates existing book's info in DB
// @Tags Books
// @Accept json
// @Produce json
// @Param request body RequestBookWithID true "Book's info"
// @Success 200 {string} string "Updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or empty title"
// @Failure 404 {object} ErrorResponse "Book not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/books [put]
func (cfg *ApiConfig) HandlePutApiBooks(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := RequestBookWithID{}
	err := decoder.Decode(&request)
	if err != nil || request.Title == "" {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	bookUUID, uuidErr := uuid.Parse(request.Id)
	if uuidErr != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	tx, err := cfg.DB.BeginTx(r.Context(), nil)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Print("Failed to rollback transaction ", rollbackErr)
			}
			common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		} else {
			err = tx.Commit()
			if err != nil {
				common.RespondWithError(w, http.StatusInternalServerError, err.Error())
			}
			w.WriteHeader(http.StatusOK)
		}
	}()

	queries := database.New(tx)

	count, err := queries.UpdateBook(r.Context(), database.UpdateBookParams{ID: bookUUID, Title: request.Title})
	if count == 0 {
		common.RespondWithError(w, http.StatusNotFound, "Unknown book id")
		return
	}
	if err != nil {
		return
	}

	err = updateBookAuthors(queries, w, r, request.Authors, bookUUID)
	w.WriteHeader(http.StatusOK)
}

// @Summary Gets book
// @Description Gets book with requested ID from DB
// @Tags Books
// @Accept json
// @Produce json
// @Param id path string true "Book ID"
// @Success 200 {object} ResponseBookFullInfo "Book's full info"
// @Success 400 {object} ErrorResponse "Invalid book ID"
// @Success 404 {object} ErrorResponse "Book not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/books/{id} [get]
func (cfg *ApiConfig) HandleGetApiBooks(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	uuid, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	tx, err := cfg.DB.BeginTx(r.Context(), nil)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil {
				log.Print("Failed to rollback transaction ", rollbackErr)
			}
			common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		} else {
			err = tx.Commit()
			if err != nil {
				common.RespondWithError(w, http.StatusInternalServerError, err.Error())
			}
			w.WriteHeader(http.StatusOK)
		}
	}()

	queries := database.New(tx)

	bookTitle, err := queries.GetBook(r.Context(), uuid)
	if err == sql.ErrNoRows {
		common.RespondWithError(w, http.StatusNotFound, "Unknown book id")
		return
	}
	if err != nil {
		return
	}

	authors, err := queries.GetBookAuthors(r.Context(), uuid)
	if err != nil {
		return
	}
	sort.Strings(authors)

	common.RespondWithJSON(w, http.StatusOK, ResponseBookFullInfo{Title: bookTitle, Authors: authors}, nil)
}

// @Summary Deletes book
// @Description Deletes a book from DB with requested ID
// @Tags Admin Books
// @Accept json
// @Produce json
// @Param id path string true "Book ID"
// @Success 200 {string} string "Deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid book ID"
// @Failure 500 {object} ErrorResponse
// @Router /admin/books/{id} [delete]
func (cfg *ApiConfig) HandleDeleteAdminBooks(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	queries := database.New(cfg.DB)
	dbErr := queries.DeleteBook(r.Context(), uuid)
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

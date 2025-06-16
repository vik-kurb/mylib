package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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

func insertBookAuthors(queries *database.Queries, w http.ResponseWriter, r *http.Request, authorIDs []string, bookID uuid.UUID) error {
	authors := make([]uuid.UUID, 0)
	for _, authorID := range authorIDs {
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

func updateBookAuthors(queries *database.Queries, w http.ResponseWriter, r *http.Request, authorIDs []string, bookID uuid.UUID) error {
	oldAuthorIDs, dbErr := queries.GetAuthorsByBook(r.Context(), bookID)
	if dbErr != nil {
		return dbErr
	}

	newAuthors := make([]uuid.UUID, 0)
	for _, authorID := range authorIDs {
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

// @Summary Create new book
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
	defer handleTx(tx, &err, w, nil)

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
	w.WriteHeader(http.StatusCreated)
}

// @Summary Update book
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

	bookUUID, uuidErr := uuid.Parse(request.ID)
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
	responseStatus := http.StatusInternalServerError
	defer handleTx(tx, &err, w, &responseStatus)

	queries := database.New(tx)

	count, err := queries.UpdateBook(r.Context(), database.UpdateBookParams{ID: bookUUID, Title: request.Title})
	if count == 0 {
		responseStatus = http.StatusNotFound
		return
	}
	if err != nil {
		return
	}

	err = updateBookAuthors(queries, w, r, request.Authors, bookUUID)
	if err != nil {
		return
	}
	w.WriteHeader(http.StatusOK)
}

// @Summary Delete book
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

	w.WriteHeader(http.StatusNoContent)
}

func parseBookIDs(r *http.Request) ([]uuid.UUID, error) {
	decoder := json.NewDecoder(r.Body)
	request := RequestBookIDs{}
	err := decoder.Decode(&request)
	if err != nil {
		return nil, err
	}
	if len(request.BookIDs) == 0 {
		return nil, errors.New("empty books list")
	}
	bookUUIDs := make([]uuid.UUID, 0)
	for _, bookID := range request.BookIDs {
		uuid, err := uuid.Parse(bookID)
		if err != nil {
			log.Print("Invalid bookID: ", bookID)
			continue
		}
		bookUUIDs = append(bookUUIDs, uuid)
	}
	return bookUUIDs, nil
}

func handleTx(tx *sql.Tx, err *error, w http.ResponseWriter, responseStatus *int) {
	if p := recover(); p != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Print("Failed to rollback transaction ", rollbackErr)
		}
		common.RespondWithError(w, http.StatusInternalServerError, fmt.Sprintf("panic recovered in handleTx: %v", p))
		return
	}
	if err != nil && *err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			log.Print("Failed to rollback transaction ", rollbackErr)
		}
		if responseStatus == nil {
			common.RespondWithError(w, http.StatusInternalServerError, (*err).Error())
			return
		}
		common.RespondWithError(w, *responseStatus, (*err).Error())
		return
	}
	if commitErr := tx.Commit(); commitErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, commitErr.Error())
		return
	}
}

func getUniqueAuthors(bookAuthors []database.GetAuthorsByBooksRow) []uuid.UUID {
	uniqueAuthors := make(map[uuid.UUID]bool)
	for _, bookAuthor := range bookAuthors {
		uniqueAuthors[bookAuthor.AuthorID] = true
	}
	authors := make([]uuid.UUID, 0, len(uniqueAuthors))
	for author := range uniqueAuthors {
		authors = append(authors, author)
	}
	return authors
}

func getBooksAndAuthors(ctx context.Context, queries *database.Queries, bookUUIDs []uuid.UUID) ([]database.GetBooksRow, map[uuid.UUID][]string, error) {
	books, err := queries.GetBooks(ctx, bookUUIDs)
	if err != nil {
		return nil, nil, err
	}

	bookAuthors, err := queries.GetAuthorsByBooks(ctx, bookUUIDs)
	if err != nil {
		return nil, nil, err
	}
	authors := getUniqueAuthors(bookAuthors)
	authorsInfo, err := queries.GetAuthorsByIDs(ctx, authors)
	if err != nil {
		return nil, nil, err
	}
	authorToName := make(map[uuid.UUID]string)
	for _, author := range authorsInfo {
		authorToName[author.ID] = author.FullName
	}
	bookToAuthors := make(map[uuid.UUID][]string)
	for _, bookAuthor := range bookAuthors {
		authorName := authorToName[bookAuthor.AuthorID]
		if authorName != "" {
			bookToAuthors[bookAuthor.BookID] = append(bookToAuthors[bookAuthor.BookID], authorName)
		}
	}
	for _, authors := range bookToAuthors {
		sort.Strings(authors)
	}

	return books, bookToAuthors, nil
}

// @Summary Get books
// @Description Gets books full info from DB
// @Tags Books
// @Accept json
// @Produce json
// @Param request body RequestBookIDs true "Book ids"
// @Success 200 {array} ResponseBookFullInfo "Books full info"
// @Failure 400 {object} ErrorResponse "Invalid request"
// @Failure 500 {object} ErrorResponse
// @Router /api/books/search [post]
func (cfg *ApiConfig) HandlePostApiBooksSearch(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	bookUUIDs, err := parseBookIDs(r)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := cfg.DB.BeginTx(r.Context(), nil)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}
	defer handleTx(tx, &err, w, nil)

	queries := database.New(tx)
	books, bookToAuthors, err := getBooksAndAuthors(r.Context(), queries, bookUUIDs)
	if err != nil {
		return
	}

	response := make([]ResponseBookFullInfo, 0, len(books))
	for _, book := range books {
		responseBook := ResponseBookFullInfo{ID: book.ID.String(), Title: book.Title}
		responseBook.Authors = bookToAuthors[book.ID]
		response = append(response, responseBook)
	}

	sort.Slice(response, func(i, j int) bool { return response[i].Title < response[j].Title })

	common.RespondWithJSON(w, http.StatusOK, response, nil)
}

// @Summary Search books by title
// @Description Searches books by title. Uses postgres full text search
// @Tags Books
// @Accept json
// @Produce json
// @Param text query string true "Search text"
// @Success 200 {array} ResponseBookFullInfo "Books' full info"
// @Success 400 {object} ErrorResponse "Empty search text"
// @Failure 500 {object} ErrorResponse
// @Router /api/books/search [get]
func (cfg *ApiConfig) HandleGetApiBooksSearch(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	searchText := r.URL.Query().Get("text")
	if searchText == "" {
		common.RespondWithError(w, http.StatusBadRequest, "Empty search text")
		return
	}

	tx, err := cfg.DB.BeginTx(r.Context(), nil)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}
	defer handleTx(tx, &err, w, nil)

	queries := database.New(tx)
	books, dbErr := queries.SearchBooks(r.Context(), database.SearchBooksParams{PlaintoTsquery: searchText, Limit: int32(cfg.MaxSearchBooksLimit)})
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	bookIDs := make([]uuid.UUID, 0, len(books))
	for _, book := range books {
		bookIDs = append(bookIDs, book.ID)
	}

	bookAuthors, dbErr := queries.GetAuthorsNamesByBooks(r.Context(), bookIDs)
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	bookToAuthors := make(map[uuid.UUID][]string)
	for _, bookAuthor := range bookAuthors {
		bookToAuthors[bookAuthor.BookID] = append(bookToAuthors[bookAuthor.BookID], bookAuthor.FullName)
	}
	for _, authors := range bookToAuthors {
		sort.Strings(authors)
	}

	responseBooks := make([]ResponseBookFullInfo, 0, len(books))
	for _, book := range books {
		responseBook := ResponseBookFullInfo{ID: book.ID.String(), Title: book.Title}
		if authors, ok := bookToAuthors[book.ID]; ok {
			responseBook.Authors = authors
		}
		responseBooks = append(responseBooks, responseBook)
	}
	common.RespondWithJSON(w, http.StatusOK, responseBooks, nil)
}

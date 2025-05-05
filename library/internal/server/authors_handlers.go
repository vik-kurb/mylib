package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/bakurvik/mylib/library/internal/database"

	common "github.com/bakurvik/mylib-common"
	"github.com/google/uuid"
)

// @Summary Ping the server
// @Description  Checks server health. Returns 200 OK if server is up.
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {string} string
// @Router /ping [get]
func (cfg *ApiConfig) HandlePing(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// @Summary Create new author
// @Description Creates new author and stores it in DB
// @Tags Authors
// @Accept json
// @Produce json
// @Param request body RequestAuthor true "Author's info"
// @Success 201 {string} string "Created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or empty full_name"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors [post]
func (cfg *ApiConfig) HandlePostApiAuthors(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := RequestAuthor{}
	err := decoder.Decode(&request)
	if err != nil || request.FullName == "" {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	queries := database.New(cfg.DB)
	dbErr := queries.CreateAuthor(
		r.Context(),
		database.CreateAuthorParams{
			FullName:  request.FullName,
			BirthDate: common.ToNullTime(request.BirthDate),
			DeathDate: common.ToNullTime(request.DeathDate)})
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// @Summary Get authors
// @Description Gets all authors from DB
// @Tags Authors
// @Accept json
// @Produce json
// @Success 200 {array} ResponseAuthorShortInfo "Author's short info"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors [get]
func (cfg *ApiConfig) HandleGetApiAuthors(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	queries := database.New(cfg.DB)
	authors, dbErr := queries.GetAuthors(r.Context())
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}

	sort.Slice(authors, func(i, j int) bool { return authors[i].FullName < authors[j].FullName })

	responseAuthors := make([]ResponseAuthorShortInfo, 0, len(authors))
	for _, author := range authors {
		responseAuthors = append(responseAuthors, ResponseAuthorShortInfo{FullName: author.FullName, Id: author.ID.String()})
	}
	common.RespondWithJSON(w, http.StatusOK, responseAuthors, nil)
}

// @Summary Get author
// @Description Gets an author with requested ID from DB
// @Tags Authors
// @Accept json
// @Produce json
// @Param id path string true "Author ID"
// @Success 200 {object} ResponseAuthorFullInfo "Author's full info"
// @Success 400 {object} ErrorResponse "Invalid author ID"
// @Failure 404 {object} ErrorResponse "Author not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors/{id} [get]
func (cfg *ApiConfig) HandleGetApiAuthorsId(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	uuid, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	queries := database.New(cfg.DB)
	author, dbErr := queries.GetAuthor(r.Context(), uuid)
	if dbErr == sql.ErrNoRows {
		common.RespondWithError(w, http.StatusNotFound, "Author not found")
		return
	}
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}

	common.RespondWithJSON(w, http.StatusOK, ResponseAuthorFullInfo{FullName: author.FullName, BirthDate: author.BirthDate.Time.Format(common.DateFormat), DeathDate: author.DeathDate.Time.Format(common.DateFormat)}, nil)
}

// @Summary Delete author
// @Description Deletes an author with requested ID from DB
// @Tags Admin Authors
// @Accept json
// @Produce json
// @Param id path string true "Author ID"
// @Success 200 {string} string "Deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid author ID"
// @Failure 500 {object} ErrorResponse
// @Router /admin/authors/{id} [delete]
func (cfg *ApiConfig) HandleDeleteAdminAuthors(w http.ResponseWriter, r *http.Request) {
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
	dbErr := queries.DeleteAuthor(r.Context(), uuid)
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary Update author
// @Description Updates existing author's info in DB
// @Tags Authors
// @Accept json
// @Produce json
// @Param request body RequestAuthor true "Author's info"
// @Success 200 {string} string "Updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body or empty full_name"
// @Failure 404 {object} ErrorResponse "Author not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors [put]
func (cfg *ApiConfig) HandlePutApiAuthors(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := RequestAuthorWithID{}
	err := decoder.Decode(&request)
	if err != nil || request.FullName == "" {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	uuid, uuidErr := uuid.Parse(request.Id)
	if uuidErr != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	queries := database.New(cfg.DB)
	rowsCount, dbErr := queries.UpdateAuthor(
		r.Context(),
		database.UpdateAuthorParams{
			ID:        uuid,
			FullName:  request.FullName,
			BirthDate: common.ToNullTime(request.BirthDate),
			DeathDate: common.ToNullTime(request.DeathDate)})
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	if rowsCount == 0 {
		common.RespondWithError(w, http.StatusNotFound, "Author not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}

// @Summary Get author's books
// @Description Returns a list of books written by the specified author
// @Tags Authors
// @Accept json
// @Produce json
// @Param id path string true "Author ID"
// @Success 200 {array} ResponseBook "Author's books"
// @Success 400 {object} ErrorResponse "Invalid author ID"
// @Failure 404 {object} ErrorResponse "Author not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors/{id}/books [get]
func (cfg *ApiConfig) HandleGetApiAuthorsBooks(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	uuid, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	queries := database.New(cfg.DB)
	_, dbErr := queries.GetAuthor(r.Context(), uuid)
	if dbErr == sql.ErrNoRows {
		common.RespondWithError(w, http.StatusNotFound, "Author not found")
		return
	}
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}

	books, dbErr := queries.GetBooksByAuthor(r.Context(), uuid)
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	response := make([]ResponseBook, 0, len(books))
	for _, book := range books {
		response = append(response, ResponseBook{Id: book.ID.String(), Title: book.Title})
	}
	sort.Slice(response, func(i, j int) bool { return response[i].Title < response[j].Title })

	common.RespondWithJSON(w, http.StatusOK, response, nil)
}

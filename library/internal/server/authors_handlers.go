package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/bakurvik/mylib/library/internal/database"

	"github.com/bakurvik/mylib/common"
	"github.com/google/uuid"
)

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
	if len(authors) == 0 {
		common.RespondWithError(w, http.StatusNotFound, "No authors")
		return
	}

	sort.Slice(authors, func(i, j int) bool { return authors[i].FullName < authors[j].FullName })

	responseAuthors := make([]ResponseAuthorShortInfo, 0, len(authors))
	for _, author := range authors {
		responseAuthors = append(responseAuthors, ResponseAuthorShortInfo{FullName: author.FullName, Id: author.ID.String()})
	}
	common.RespondWithJSON(w, http.StatusOK, responseAuthors, nil)
}

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

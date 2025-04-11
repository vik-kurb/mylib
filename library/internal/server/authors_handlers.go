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
	type requestBody struct {
		FullName  string `json:"full_name"`
		BirthDate string `json:"birth_date,omitempty"`
		DeathDate string `json:"death_date,omitempty"`
	}
	request := requestBody{}
	err := decoder.Decode(&request)
	if err != nil || request.FullName == "" {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	db_err := cfg.DB.CreateAuthor(
		r.Context(),
		database.CreateAuthorParams{
			FullName:  request.FullName,
			BirthDate: common.ToNullTime(request.BirthDate),
			DeathDate: common.ToNullTime(request.DeathDate)})
	if db_err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (cfg *ApiConfig) HandleGetApiAuthors(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	authors, db_err := cfg.DB.GetAuthors(r.Context())
	if db_err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}
	if len(authors) == 0 {
		common.RespondWithError(w, http.StatusNotFound, "No authors")
		return
	}

	sort.Slice(authors, func(i, j int) bool { return authors[i].FullName < authors[j].FullName })

	type responseAuthor struct {
		FullName string `json:"full_name"`
		Id       string `json:"id"`
	}
	response_authors := make([]responseAuthor, 0, len(authors))
	for _, author := range authors {
		response_authors = append(response_authors, responseAuthor{FullName: author.FullName, Id: author.ID.String()})
	}
	common.RespondWithJSON(w, http.StatusOK, response_authors)
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
	author, db_err := cfg.DB.GetAuthor(r.Context(), uuid)
	if db_err == sql.ErrNoRows {
		common.RespondWithError(w, http.StatusNotFound, "Author not found")
		return
	}
	if db_err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}

	type responseAuthor struct {
		FullName  string `json:"full_name"`
		BirthDate string `json:"birth_date,omitempty"`
		DeathDate string `json:"death_date,omitempty"`
	}
	common.RespondWithJSON(w, http.StatusOK, responseAuthor{FullName: author.FullName, BirthDate: author.BirthDate.Time.Format(common.DateFormat), DeathDate: author.DeathDate.Time.Format(common.DateFormat)})
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

	db_err := cfg.DB.DeleteAuthor(r.Context(), uuid)
	if db_err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (cfg *ApiConfig) HandlePutApiAuthors(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	type requestBody struct {
		Id        string `json:"id"`
		FullName  string `json:"full_name"`
		BirthDate string `json:"birth_date,omitempty"`
		DeathDate string `json:"death_date,omitempty"`
	}
	request := requestBody{}
	err := decoder.Decode(&request)
	if err != nil || request.FullName == "" {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	uuid, uuid_err := uuid.Parse(request.Id)
	if uuid_err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	rows_count, db_err := cfg.DB.UpdateAuthor(
		r.Context(),
		database.UpdateAuthorParams{
			ID:        uuid,
			FullName:  request.FullName,
			BirthDate: common.ToNullTime(request.BirthDate),
			DeathDate: common.ToNullTime(request.DeathDate)})
	if db_err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}
	if rows_count == 0 {
		common.RespondWithError(w, http.StatusNotFound, "Author not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}

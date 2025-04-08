package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"library/internal/database"
	"net/http"
	"sort"
	"time"

	"github.com/google/uuid"
)

const (
	kDateFormat = "02.01.2006"
)

func ToNullTime(s string) sql.NullTime {
	if s == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse(kDateFormat, s)
	if err != nil {
		fmt.Println("Error while parsing date:", err)
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func (cfg *ApiConfig) HandlePostApiAuthors(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	type requestBody struct {
		FirstName  string `json:"first_name"`
		FamilyName string `json:"family_name"`
		BirthDate  string `json:"birth_date,omitempty"`
		DeathDate  string `json:"death_date,omitempty"`
	}
	request := requestBody{}
	err := decoder.Decode(&request)
	if err != nil || request.FamilyName == "" || request.FirstName == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if cfg.DB == nil {
		respondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	db_err := cfg.DB.CreateAuthor(
		r.Context(),
		database.CreateAuthorParams{
			FirstName:  request.FirstName,
			FamilyName: request.FamilyName,
			BirthDate:  ToNullTime(request.BirthDate),
			DeathDate:  ToNullTime(request.DeathDate)})
	if db_err != nil {
		respondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func buildName(first_name, family_name string) string {
	return fmt.Sprintf("%v %v", first_name, family_name)
}

func (cfg *ApiConfig) HandleGetApiAuthors(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		respondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	authors, db_err := cfg.DB.GetAuthors(r.Context())
	if db_err != nil {
		respondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}
	if len(authors) == 0 {
		respondWithError(w, http.StatusNotFound, "No authors")
		return
	}

	sort.Slice(authors, func(i, j int) bool { return authors[i].FamilyName < authors[j].FamilyName })

	type responseAuthor struct {
		Name string `json:"name"`
		Id   string `json:"id"`
	}
	response_authors := make([]responseAuthor, 0, len(authors))
	for _, author := range authors {
		response_authors = append(response_authors, responseAuthor{Name: buildName(author.FirstName, author.FamilyName), Id: author.ID.String()})
	}
	respondWithJSON(w, http.StatusOK, response_authors)
}

func (cfg *ApiConfig) HandleGetApiAuthorsId(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		respondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	uuid, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}
	author, db_err := cfg.DB.GetAuthor(r.Context(), uuid)
	if db_err == sql.ErrNoRows {
		respondWithError(w, http.StatusNotFound, "Author not found")
		return
	}
	if db_err != nil {
		respondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}

	type responseAuthor struct {
		FirstName  string `json:"first_name"`
		FamilyName string `json:"family_name"`
		BirthDate  string `json:"birth_date,omitempty"`
		DeathDate  string `json:"death_date,omitempty"`
	}
	respondWithJSON(w, http.StatusOK, responseAuthor{FirstName: author.FirstName, FamilyName: author.FamilyName, BirthDate: author.BirthDate.Time.Format(kDateFormat), DeathDate: author.DeathDate.Time.Format(kDateFormat)})
}

func (cfg *ApiConfig) HandleDeleteAdminAuthors(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if cfg.DB == nil {
		respondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	db_err := cfg.DB.DeleteAuthor(r.Context(), uuid)
	if db_err != nil {
		respondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (cfg *ApiConfig) HandlePutApiAuthors(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	type requestBody struct {
		Id         string `json:"id"`
		FirstName  string `json:"first_name"`
		FamilyName string `json:"family_name"`
		BirthDate  string `json:"birth_date,omitempty"`
		DeathDate  string `json:"death_date,omitempty"`
	}
	request := requestBody{}
	err := decoder.Decode(&request)
	if err != nil || request.FamilyName == "" || request.FirstName == "" {
		respondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	uuid, uuid_err := uuid.Parse(request.Id)
	if uuid_err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid id")
		return
	}

	if cfg.DB == nil {
		respondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	rows_count, db_err := cfg.DB.UpdateAuthor(
		r.Context(),
		database.UpdateAuthorParams{
			ID:         uuid,
			FirstName:  request.FirstName,
			FamilyName: request.FamilyName,
			BirthDate:  ToNullTime(request.BirthDate),
			DeathDate:  ToNullTime(request.DeathDate)})
	if db_err != nil {
		respondWithError(w, http.StatusInternalServerError, db_err.Error())
		return
	}
	if rows_count == 0 {
		respondWithError(w, http.StatusNotFound, "Author not found")
		return
	}
	w.WriteHeader(http.StatusOK)
}

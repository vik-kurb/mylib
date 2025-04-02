package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"mylib/internal/database"
	"net/http"
	"time"
)

func ToNullTime(s string) sql.NullTime {
	if s == "" {
		return sql.NullTime{}
	}
	const template = "02.01.2006"
	t, err := time.Parse(template, s)
	if err != nil {
		fmt.Println("Error while parsing date:", err)
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func (cfg *ApiConfig) HandleApiAuthors(w http.ResponseWriter, r *http.Request) {
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

package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
	"users/internal/auth"
	"users/internal/database"

	"log"
)

const (
	dateFormat = "02.01.2006"
)

type userRequestBody struct {
	LoginName string `json:"login_name"`
	Email     string `json:"email"`
	BirthDate string `json:"birth_date,omitempty"`
	Password  string `json:"password"`
}

func ToNullTime(s string) sql.NullTime {
	if s == "" {
		return sql.NullTime{}
	}
	t, err := time.Parse(dateFormat, s)
	if err != nil {
		log.Print("Error while parsing date:", err)
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

func validateUserData(request userRequestBody) error {
	const minLoginLen = 3
	if len(request.LoginName) < minLoginLen {
		return errors.New("login name is too short")
	}
	if !strings.Contains(request.Email, "@") {
		return errors.New("invalid email")
	}
	const minPasswordLen = 5
	if len(request.Password) < minPasswordLen {
		return errors.New("password is too short")
	}
	return nil
}

func (cfg *ApiConfig) HandlePostApiUsers(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := userRequestBody{}
	requestErr := decoder.Decode(&request)
	if requestErr != nil {
		respondWithError(w, http.StatusBadRequest, requestErr.Error())
		return
	}
	validateErr := validateUserData(request)
	if validateErr != nil {
		respondWithError(w, http.StatusBadRequest, validateErr.Error())
		return
	}

	rows, getUserErr := cfg.DB.GetUser(r.Context(), database.GetUserParams{LoginName: request.LoginName, Email: request.Email})
	if getUserErr != nil {
		respondWithError(w, http.StatusInternalServerError, getUserErr.Error())
		return
	}
	for _, row := range rows {
		if row.Email == request.Email {
			respondWithError(w, http.StatusConflict, "User with this email already exists")
			return
		}
		if row.LoginName == request.LoginName {
			respondWithError(w, http.StatusConflict, "User with this login already exists")
			return
		}
	}

	hashedPassword, hashErr := auth.HashPassword(request.Password)
	if hashErr != nil {
		respondWithError(w, http.StatusInternalServerError, hashErr.Error())
		return
	}

	user_id, userErr := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{LoginName: request.LoginName, Email: request.Email, BirthDate: ToNullTime(request.BirthDate), HashedPassword: hashedPassword})
	if userErr != nil {
		respondWithError(w, http.StatusInternalServerError, userErr.Error())
		return
	}

	const tokenExpiresIn = time.Hour
	accessToken, accessTokenErr := auth.MakeJWT(user_id, cfg.AuthSecretKey, tokenExpiresIn)
	if accessTokenErr != nil {
		respondWithError(w, http.StatusInternalServerError, accessTokenErr.Error())
		return
	}

	refreshToken, refreshTokenErr := auth.MakeRefreshToken()
	if refreshTokenErr != nil {
		respondWithError(w, http.StatusInternalServerError, refreshTokenErr.Error())
		return
	}
	const refreshTokenExpiresIn = 30 * 24 * time.Hour
	expiresAt := time.Now().Add(refreshTokenExpiresIn)
	saveTokenErr := cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: user_id, ExpiresAt: expiresAt})
	if saveTokenErr != nil {
		respondWithError(w, http.StatusInternalServerError, saveTokenErr.Error())
		return
	}

	type responseBody struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	const refreshTokenName = "refresh_token"
	cookie := http.Cookie{
		Name:     refreshTokenName,
		Value:    refreshToken,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	respondWithJSON(w, http.StatusCreated, responseBody{ID: user_id.String(), Token: accessToken}, &cookie)
}

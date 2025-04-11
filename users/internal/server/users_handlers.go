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

	"github.com/google/uuid"
)

const (
	dateFormat            = "02.01.2006"
	tokenExpiresIn        = time.Hour
	refreshTokenExpiresIn = 30 * 24 * time.Hour
	refreshTokenName      = "refresh_token"
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

func validateUserData(cfg *ApiConfig, r *http.Request, requestBody userRequestBody) (int, error) {
	const minLoginLen = 3
	if len(requestBody.LoginName) < minLoginLen {
		return http.StatusBadRequest, errors.New("login name is too short")
	}
	if !strings.Contains(requestBody.Email, "@") {
		return http.StatusBadRequest, errors.New("invalid email")
	}
	const minPasswordLen = 5
	if len(requestBody.Password) < minPasswordLen {
		return http.StatusBadRequest, errors.New("password is too short")
	}

	rows, getUserErr := cfg.DB.GetUser(r.Context(), database.GetUserParams{LoginName: requestBody.LoginName, Email: requestBody.Email})
	if getUserErr != nil {
		return http.StatusInternalServerError, getUserErr
	}
	for _, row := range rows {
		if row.Email == requestBody.Email {
			return http.StatusConflict, errors.New("user with this email already exists")
		}
		if row.LoginName == requestBody.LoginName {
			return http.StatusConflict, errors.New("user with this login already exists")
		}
	}

	return 0, nil
}

func makeTokensAndRespond(w http.ResponseWriter, r *http.Request, cfg *ApiConfig, userID uuid.UUID, status int) {
	accessToken, accessTokenErr := auth.MakeJWT(userID, cfg.AuthSecretKey, tokenExpiresIn)
	if accessTokenErr != nil {
		respondWithError(w, http.StatusInternalServerError, accessTokenErr.Error())
		return
	}

	expiresAt := time.Now().Add(refreshTokenExpiresIn)
	refreshToken, refreshTokenErr := auth.MakeRefreshToken()
	if refreshTokenErr != nil {
		respondWithError(w, http.StatusInternalServerError, refreshTokenErr.Error())
		return
	}
	saveTokenErr := cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: userID, ExpiresAt: expiresAt})
	if saveTokenErr != nil {
		respondWithError(w, http.StatusInternalServerError, saveTokenErr.Error())
		return
	}

	type responseBody struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	cookie := http.Cookie{
		Name:     refreshTokenName,
		Value:    refreshToken,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	respondWithJSON(w, status, responseBody{ID: userID.String(), Token: accessToken}, &cookie)
}

func revokeRefreshToken(cfg *ApiConfig, r *http.Request, refreshToken string) {
	err := cfg.DB.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Print("Failed to revoke refresh token: ", err)
	}
}

func (cfg *ApiConfig) HandlePostApiUsers(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := userRequestBody{}
	requestErr := decoder.Decode(&request)
	if requestErr != nil {
		respondWithError(w, http.StatusBadRequest, requestErr.Error())
		return
	}
	code, validateErr := validateUserData(cfg, r, request)
	if validateErr != nil {
		respondWithError(w, code, validateErr.Error())
		return
	}

	hashedPassword, hashErr := auth.HashPassword(request.Password)
	if hashErr != nil {
		respondWithError(w, http.StatusInternalServerError, hashErr.Error())
		return
	}

	userID, userErr := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{LoginName: request.LoginName, Email: request.Email, BirthDate: ToNullTime(request.BirthDate), HashedPassword: hashedPassword})
	if userErr != nil {
		respondWithError(w, http.StatusInternalServerError, userErr.Error())
		return
	}

	makeTokensAndRespond(w, r, cfg, userID, http.StatusCreated)
}

func (cfg *ApiConfig) HandlePostApiLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	type requestBody struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	request := requestBody{}
	requestErr := decoder.Decode(&request)
	if requestErr != nil {
		respondWithError(w, http.StatusBadRequest, requestErr.Error())
		return
	}

	user, getUserErr := cfg.DB.GetUserByEmail(r.Context(), request.Email)
	if getUserErr != nil {
		if getUserErr == sql.ErrNoRows {
			respondWithError(w, http.StatusNotFound, "Not found user")
			return
		}
		respondWithError(w, http.StatusInternalServerError, getUserErr.Error())
		return
	}

	checkPasswErr := auth.CheckPasswordHash(user.HashedPassword, request.Password)
	if checkPasswErr != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	makeTokensAndRespond(w, r, cfg, user.ID, http.StatusOK)
}

func (cfg *ApiConfig) HandlePostApiRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, cookieErr := r.Cookie(refreshTokenName)
	if cookieErr != nil || cookie == nil {
		respondWithError(w, http.StatusUnauthorized, "No refresh token in cookie")
		return
	}
	refreshToken := cookie.Value

	user_id, getUserErr := cfg.DB.GetUserByRefreshToken(r.Context(), refreshToken)
	if getUserErr != nil {
		if getUserErr == sql.ErrNoRows {
			respondWithError(w, http.StatusUnauthorized, "Not found user")
			return
		}
		respondWithError(w, http.StatusInternalServerError, getUserErr.Error())
		return
	}

	revokeRefreshToken(cfg, r, refreshToken)

	makeTokensAndRespond(w, r, cfg, user_id, http.StatusOK)
}

func (cfg *ApiConfig) HandlePostApiRevoke(w http.ResponseWriter, r *http.Request) {
	cookie, cookieErr := r.Cookie(refreshTokenName)
	if cookieErr != nil || cookie == nil {
		respondWithError(w, http.StatusUnauthorized, "No refresh token in cookie")
		return
	}

	revokeRefreshToken(cfg, r, cookie.Value)
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *ApiConfig) HandlePutApiUsers(w http.ResponseWriter, r *http.Request) {
	token, tokenErr := auth.GetBearerToken(r.Header)
	if tokenErr != nil {
		respondWithError(w, http.StatusUnauthorized, tokenErr.Error())
		return
	}
	userID, authErr := auth.ValidateJWT(token, cfg.AuthSecretKey)
	if authErr != nil {
		respondWithError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	decoder := json.NewDecoder(r.Body)
	request := userRequestBody{}
	requestErr := decoder.Decode(&request)
	if requestErr != nil {
		respondWithError(w, http.StatusBadRequest, requestErr.Error())
		return
	}
	code, validateErr := validateUserData(cfg, r, request)
	if validateErr != nil {
		respondWithError(w, code, validateErr.Error())
		return
	}

	hashedPassword, hashErr := auth.HashPassword(request.Password)
	if hashErr != nil {
		respondWithError(w, http.StatusInternalServerError, hashErr.Error())
		return
	}

	_, userErr := cfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{ID: userID, LoginName: request.LoginName, Email: request.Email, BirthDate: ToNullTime(request.BirthDate), HashedPassword: hashedPassword})
	if userErr != nil {
		respondWithError(w, http.StatusNotFound, userErr.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getAuthUser(cfg *ApiConfig, r *http.Request) *uuid.UUID {
	token, tokenErr := auth.GetBearerToken(r.Header)
	if tokenErr != nil {
		return nil
	}
	userID, authErr := auth.ValidateJWT(token, cfg.AuthSecretKey)
	if authErr != nil {
		return nil
	}
	return &userID
}

func (cfg *ApiConfig) HandleGetApiUsers(w http.ResponseWriter, r *http.Request) {
	requestUserID := r.PathValue("userID")
	if len(requestUserID) == 0 {
		respondWithError(w, http.StatusBadRequest, "No user id in request")
		return
	}
	requestUserUUID, err := uuid.Parse(requestUserID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	authUserIdPtr := getAuthUser(cfg, r)
	cfg.DB.GetUser(r.Context(), database.GetUserParams{})

	user, userErr := cfg.DB.GetUserById(r.Context(), requestUserUUID)
	if userErr != nil {
		respondWithError(w, http.StatusNotFound, userErr.Error())
		return
	}

	type responseBody struct {
		LoginName string `json:"login"`
		Email     string `json:"email"`
		BirthDate string `json:"birth_date"`
	}
	response := responseBody{LoginName: user.LoginName}
	if authUserIdPtr != nil && *authUserIdPtr == requestUserUUID {
		if user.BirthDate.Valid {
			response.BirthDate = user.BirthDate.Time.Format(dateFormat)
		}
		response.Email = user.Email
	}
	respondWithJSON(w, http.StatusOK, response, nil)
}

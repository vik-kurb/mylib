package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bakurvik/mylib/users/internal/database"

	"github.com/bakurvik/mylib/users/internal/auth"

	"log"

	common "github.com/bakurvik/mylib-common"
	"github.com/google/uuid"
)

const (
	tokenExpiresIn        = time.Hour
	refreshTokenExpiresIn = 30 * 24 * time.Hour
	refreshTokenName      = "refresh_token"
)

func validateUserData(cfg *ApiConfig, r *http.Request, requestBody RequestUser) (int, error) {
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
		common.RespondWithError(w, http.StatusInternalServerError, accessTokenErr.Error())
		return
	}

	expiresAt := time.Now().Add(refreshTokenExpiresIn)
	refreshToken, refreshTokenErr := auth.MakeRefreshToken()
	if refreshTokenErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, refreshTokenErr.Error())
		return
	}
	saveTokenErr := cfg.DB.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{Token: refreshToken, UserID: userID, ExpiresAt: expiresAt})
	if saveTokenErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, saveTokenErr.Error())
		return
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
	common.RespondWithJSON(w, status, ResponseToken{ID: userID.String(), Token: accessToken}, &cookie)
}

func revokeRefreshToken(cfg *ApiConfig, r *http.Request, refreshToken string) {
	err := cfg.DB.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		log.Print("Failed to revoke refresh token: ", err)
	}
}

func checkAuthorization(cfg *ApiConfig, r *http.Request) (uuid.UUID, error) {
	token, tokenErr := auth.GetBearerToken(r.Header)
	if tokenErr != nil {
		return uuid.UUID{}, tokenErr
	}
	userID, authErr := auth.ValidateJWT(token, cfg.AuthSecretKey)
	if authErr != nil {
		return uuid.UUID{}, authErr
	}
	return userID, nil
}

func (cfg *ApiConfig) HandlePostApiUsers(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := RequestUser{}
	requestErr := decoder.Decode(&request)
	if requestErr != nil {
		common.RespondWithError(w, http.StatusBadRequest, requestErr.Error())
		return
	}
	code, validateErr := validateUserData(cfg, r, request)
	if validateErr != nil {
		common.RespondWithError(w, code, validateErr.Error())
		return
	}

	hashedPassword, hashErr := auth.HashPassword(request.Password)
	if hashErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, hashErr.Error())
		return
	}

	userID, userErr := cfg.DB.CreateUser(r.Context(), database.CreateUserParams{LoginName: request.LoginName, Email: request.Email, BirthDate: common.ToNullTime(request.BirthDate), HashedPassword: hashedPassword})
	if userErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, userErr.Error())
		return
	}

	makeTokensAndRespond(w, r, cfg, userID, http.StatusCreated)
}

func (cfg *ApiConfig) HandlePostApiLogin(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	request := RequestLogin{}
	requestErr := decoder.Decode(&request)
	if requestErr != nil {
		common.RespondWithError(w, http.StatusBadRequest, requestErr.Error())
		return
	}

	user, getUserErr := cfg.DB.GetUserByEmail(r.Context(), request.Email)
	if getUserErr != nil {
		if getUserErr == sql.ErrNoRows {
			common.RespondWithError(w, http.StatusNotFound, "Not found user")
			return
		}
		common.RespondWithError(w, http.StatusInternalServerError, getUserErr.Error())
		return
	}

	checkPasswErr := auth.CheckPasswordHash(user.HashedPassword, request.Password)
	if checkPasswErr != nil {
		common.RespondWithError(w, http.StatusUnauthorized, "Invalid password")
		return
	}

	makeTokensAndRespond(w, r, cfg, user.ID, http.StatusOK)
}

func (cfg *ApiConfig) HandlePostApiRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, cookieErr := r.Cookie(refreshTokenName)
	if cookieErr != nil || cookie == nil {
		common.RespondWithError(w, http.StatusUnauthorized, "No refresh token in cookie")
		return
	}
	refreshToken := cookie.Value

	userID, getUserErr := cfg.DB.GetUserByRefreshToken(r.Context(), refreshToken)
	if getUserErr != nil {
		if getUserErr == sql.ErrNoRows {
			common.RespondWithError(w, http.StatusUnauthorized, "Not found user")
			return
		}
		common.RespondWithError(w, http.StatusInternalServerError, getUserErr.Error())
		return
	}

	revokeRefreshToken(cfg, r, refreshToken)

	makeTokensAndRespond(w, r, cfg, userID, http.StatusOK)
}

func (cfg *ApiConfig) HandlePostApiRevoke(w http.ResponseWriter, r *http.Request) {
	cookie, cookieErr := r.Cookie(refreshTokenName)
	if cookieErr != nil || cookie == nil {
		common.RespondWithError(w, http.StatusUnauthorized, "No refresh token in cookie")
		return
	}

	revokeRefreshToken(cfg, r, cookie.Value)
	w.WriteHeader(http.StatusNoContent)
}

func (cfg *ApiConfig) HandlePutApiUsers(w http.ResponseWriter, r *http.Request) {
	userID, authErr := checkAuthorization(cfg, r)
	if authErr != nil {
		common.RespondWithError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	decoder := json.NewDecoder(r.Body)
	request := RequestUser{}
	requestErr := decoder.Decode(&request)
	if requestErr != nil {
		common.RespondWithError(w, http.StatusBadRequest, requestErr.Error())
		return
	}
	code, validateErr := validateUserData(cfg, r, request)
	if validateErr != nil {
		common.RespondWithError(w, code, validateErr.Error())
		return
	}

	hashedPassword, hashErr := auth.HashPassword(request.Password)
	if hashErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, hashErr.Error())
		return
	}

	_, userErr := cfg.DB.UpdateUser(r.Context(), database.UpdateUserParams{ID: userID, LoginName: request.LoginName, Email: request.Email, BirthDate: common.ToNullTime(request.BirthDate), HashedPassword: hashedPassword})
	if userErr != nil {
		common.RespondWithError(w, http.StatusNotFound, userErr.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *ApiConfig) HandleGetApiUsers(w http.ResponseWriter, r *http.Request) {
	requestUserID := r.PathValue("userID")
	if len(requestUserID) == 0 {
		common.RespondWithError(w, http.StatusBadRequest, "No user id in request")
		return
	}
	requestUserUUID, err := uuid.Parse(requestUserID)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid user id")
		return
	}

	authUserID, _ := checkAuthorization(cfg, r)

	user, userErr := cfg.DB.GetUserById(r.Context(), requestUserUUID)
	if userErr != nil {
		common.RespondWithError(w, http.StatusNotFound, userErr.Error())
		return
	}

	response := ResponseUser{LoginName: user.LoginName}
	if authUserID == requestUserUUID {
		if user.BirthDate.Valid {
			response.BirthDate = user.BirthDate.Time.Format(common.DateFormat)
		}
		response.Email = user.Email
	}
	common.RespondWithJSON(w, http.StatusOK, response, nil)
}

func (cfg *ApiConfig) HandleDeleteApiUsers(w http.ResponseWriter, r *http.Request) {
	userID, authErr := checkAuthorization(cfg, r)
	if authErr != nil {
		common.RespondWithError(w, http.StatusUnauthorized, authErr.Error())
		return
	}

	deleteErr := cfg.DB.DeleteUser(r.Context(), userID)
	if deleteErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, deleteErr.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

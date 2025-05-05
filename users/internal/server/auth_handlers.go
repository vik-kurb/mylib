package server

import (
	"database/sql"
	"encoding/json"
	"net/http"
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

// @Summary Login user
// @Description Checks password and returns access and refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RequestLogin true "User's login data"
// @Success 200 {object} ResponseToken "Logined successfully"
// @Header 200 {string} Set-Cookie "HTTP-only cookie named refresh_token"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Invalid password"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/login [post]
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

// @Summary Refresh tokens
// @Description Checks refresh token from an HTTP-only cookie and returns new access and refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} ResponseToken "Refreshed tokens successfully"
// @Header 200 {string} Set-Cookie "HTTP-only cookie named refresh_token"
// @Failure 401 {object} ErrorResponse "Invalid refresh token"
// @Failure 500 {object} ErrorResponse
// @Router /api/refresh [post]
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

// @Summary Revoke token
// @Description Revokes refresh token from an HTTP-only cookie
// @Tags Auth
// @Accept json
// @Produce json
// @Success 204
// @Failure 401 {object} ErrorResponse "No refresh token in cookie"
// @Failure 500 {object} ErrorResponse
// @Router /api/revoke [post]
func (cfg *ApiConfig) HandlePostApiRevoke(w http.ResponseWriter, r *http.Request) {
	cookie, cookieErr := r.Cookie(refreshTokenName)
	if cookieErr != nil || cookie == nil {
		common.RespondWithError(w, http.StatusUnauthorized, "No refresh token in cookie")
		return
	}

	revokeRefreshToken(cfg, r, cookie.Value)
	w.WriteHeader(http.StatusNoContent)
}

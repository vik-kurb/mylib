package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/bakurvik/mylib/users/internal/database"

	"github.com/bakurvik/mylib/users/internal/auth"

	common "github.com/bakurvik/mylib-common"
	"github.com/google/uuid"
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

// @Summary Create new user
// @Description Creates new user and stores it in DB
// @Tags Users
// @Accept json
// @Produce json
// @Param request body RequestUser true "User's info"
// @Success 201 {object} ResponseToken "Created successfully"
// @Header 201 {string} Set-Cookie "HTTP-only cookie named refresh_token"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 409 {object} ErrorResponse "User with request info already exists"
// @Failure 500 {object} ErrorResponse
// @Router /api/users [post]
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

// @Summary Update user
// @Description Updates existing user's info in DB. Uses refresh token from an HTTP-only cookie
// @Tags Users
// @Accept json
// @Produce json
// @Param request body RequestUser true "User's info"
// @Success 204
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/users [put]
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

// @Summary Get user info
// @Description Gets user from DB
// @Tags Users
// @Accept json
// @Produce json
// @Param userID path string true "Author ID"
// @Success 200 {object} ResponseUser "User's full info"
// @Success 400 {object} ErrorResponse "Invalid user ID"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 404 {object} ErrorResponse "User not found"
// @Failure 500 {object} ErrorResponse
// @Router /api/users/{id} [get]
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

// @Summary Delete user
// @Description Deletes user from DB. Uses refresh token from an HTTP-only cookie
// @Tags Users
// @Accept json
// @Produce json
// @Success 204
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/users [delete]
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

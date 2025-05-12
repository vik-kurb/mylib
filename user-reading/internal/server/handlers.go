package server

import (
	"encoding/json"
	"net/http"
	"user-reading/internal/clients"

	"github.com/bakurvik/mylib/user-reading/internal/database"

	common "github.com/bakurvik/mylib-common"
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

// @Summary Add user reading
// @Description Saves book to user reading in DB. Uses access token from an HTTP-only cookie
// @Tags User reading
// @Accept json
// @Produce json
// @Param request body RequestUserReading true "Book id with status"
// @Success 201 {string} string "Created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors [post]
func (cfg *ApiConfig) HandlePostApiUserReadingPath(w http.ResponseWriter, r *http.Request) {
	//
	decoder := json.NewDecoder(r.Body)
	request := RequestUserReading{}
	err := decoder.Decode(&request)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	userID, statusCode, err := clients.GetUser(r.Header, cfg.UsersServiceHost, cfg.UsersServicePort)
	if statusCode == http.StatusUnauthorized {
		common.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to check authorization")
		return
	}

	queries := database.New(cfg.DB)
	dbErr := queries.CreateUserReading(
		r.Context(),
		database.CreateUserReadingParams{
			userID: userID,
			bookID: request.BookId,
			status: request.Status})
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

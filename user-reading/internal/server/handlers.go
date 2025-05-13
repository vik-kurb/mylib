package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bakurvik/mylib/user-reading/internal/clients"
	"github.com/bakurvik/mylib/user-reading/internal/database"
	"github.com/google/uuid"

	common "github.com/bakurvik/mylib-common"
)

func mapUserReadingStatus(status string) (database.ReadingStatus, error) {
	dbStatus := database.ReadingStatus(status)
	if dbStatus == database.ReadingStatusFinished || dbStatus == database.ReadingStatusReading || dbStatus == database.ReadingStatusWantToRead {
		return dbStatus, nil
	}
	return "", errors.New("unknown reading status")
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
	decoder := json.NewDecoder(r.Body)
	request := RequestUserReading{}
	err := decoder.Decode(&request)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid request")
		return
	}
	bookUUID, err := uuid.Parse(request.BookId)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid book id")
		return
	}
	readingStatus, err := mapUserReadingStatus(request.Status)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	type userResponse struct {
		userID     uuid.UUID
		statusCode int
		err        error
	}
	userChan := make(chan userResponse)
	go func() {
		userID, statusCode, err := clients.GetUser(r.Header, cfg.UsersServiceHost)
		userChan <- userResponse{userID: userID, statusCode: statusCode, err: err}
	}()

	type bookResponse struct {
		statusCode int
		err        error
	}
	bookChan := make(chan bookResponse)
	go func() {
		statusCode, err := clients.CheckBook(bookUUID, cfg.LibraryServiceHost)
		bookChan <- bookResponse{statusCode: statusCode, err: err}
	}()

	userResp := <-userChan
	bookResp := <-bookChan
	if userResp.statusCode == http.StatusUnauthorized {
		common.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if userResp.err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to check authorization: "+userResp.err.Error())
		return
	}

	switch bookResp.statusCode {
	case http.StatusNotFound:
		common.RespondWithError(w, http.StatusBadRequest, "Book not found")
		return
	case http.StatusBadRequest:
		common.RespondWithError(w, http.StatusBadRequest, "Invalid book id")
		return
	case http.StatusInternalServerError:
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to check book")
		return
	}
	if userResp.err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to check book: "+userResp.err.Error())
		return
	}

	queries := database.New(cfg.DB)
	dbErr := queries.CreateUserReading(
		r.Context(),
		database.CreateUserReadingParams{
			UserID: userResp.userID,
			BookID: bookUUID,
			Status: readingStatus})
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bakurvik/mylib/user-reading/internal/clients"
	"github.com/bakurvik/mylib/user-reading/internal/database"
	"github.com/google/uuid"

	common "github.com/bakurvik/mylib-common"
)

type parsedUserReading struct {
	bookID     uuid.UUID
	status     database.ReadingStatus
	rating     int32
	startDate  sql.NullTime
	finishDate sql.NullTime
}

type bookReadInfo struct {
	status string
	rating int
}

func mapUserReadingStatus(status string) (database.ReadingStatus, error) {
	dbStatus := database.ReadingStatus(status)
	if dbStatus == database.ReadingStatusFinished || dbStatus == database.ReadingStatusReading || dbStatus == database.ReadingStatusWantToRead {
		return dbStatus, nil
	}
	return "", errors.New("unknown reading status")
}

func parseUserReading(r *http.Request) (parsedUserReading, error) {
	decoder := json.NewDecoder(r.Body)
	request := UserReading{}
	err := decoder.Decode(&request)
	if err != nil {
		return parsedUserReading{}, err
	}
	bookUUID, err := uuid.Parse(request.BookID)
	if err != nil {
		return parsedUserReading{}, err
	}
	readingStatus, err := mapUserReadingStatus(request.Status)
	if err != nil {
		return parsedUserReading{}, err
	}
	startDate := common.ToNullTime(request.StartDate)
	finishDate := common.ToNullTime(request.FinishDate)
	if startDate.Valid && finishDate.Valid && startDate.Time.After(finishDate.Time) {
		return parsedUserReading{}, errors.New("Invalid start and finish dates")
	}
	res := parsedUserReading{bookID: bookUUID, status: readingStatus, startDate: startDate, finishDate: finishDate}
	if readingStatus == database.ReadingStatusFinished {
		res.rating = int32(request.Rating)
	}
	return res, nil
}

func checkUserAndBook(r *http.Request, cfg *ApiConfig, bookUUID uuid.UUID) (uuid.UUID, int, error) {
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
		return uuid.Nil, http.StatusUnauthorized, errors.New("Unauthorized")
	}
	if userResp.err != nil {
		return uuid.Nil, http.StatusInternalServerError, errors.New("failed to check authorization")
	}

	switch bookResp.statusCode {
	case http.StatusNotFound:
		return uuid.Nil, http.StatusBadRequest, errors.New("book not found")
	case http.StatusBadRequest:
		return uuid.Nil, http.StatusBadRequest, errors.New("invalid book id")
	case http.StatusInternalServerError:
		return uuid.Nil, http.StatusInternalServerError, errors.New("failed to check book")
	}
	if bookResp.err != nil {
		return uuid.Nil, http.StatusInternalServerError, errors.New("failed to check book")
	}
	return userResp.userID, http.StatusOK, nil
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
// @Param request body UserReading true "Book id with status"
// @Success 201 {string} string "Created successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors [post]
func (cfg *ApiConfig) HandlePostApiUserReadingPath(w http.ResponseWriter, r *http.Request) {
	userReading, err := parseUserReading(r)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	userUUID, statusCode, err := checkUserAndBook(r, cfg, userReading.bookID)
	if err != nil {
		common.RespondWithError(w, statusCode, err.Error())
		return
	}

	queries := database.New(cfg.DB)
	dbErr := queries.CreateUserReading(
		r.Context(),
		database.CreateUserReadingParams{
			UserID:     userUUID,
			BookID:     userReading.bookID,
			Status:     userReading.status,
			Rating:     userReading.rating,
			StartDate:  userReading.startDate,
			FinishDate: userReading.finishDate})
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	w.WriteHeader(http.StatusCreated)
}

// @Summary Update user reading
// @Description Updates user reading in DB. Uses access token from an HTTP-only cookie
// @Tags User reading
// @Accept json
// @Produce json
// @Param request body UserReading true "Book id with status"
// @Success 201 {string} string "Updated successfully"
// @Failure 400 {object} ErrorResponse "Invalid request body"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors [post]
func (cfg *ApiConfig) HandlePutApiUserReadingPath(w http.ResponseWriter, r *http.Request) {
	userReading, err := parseUserReading(r)
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, err.Error())
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	userUUID, statusCode, err := checkUserAndBook(r, cfg, userReading.bookID)
	if err != nil {
		common.RespondWithError(w, statusCode, err.Error())
		return
	}

	queries := database.New(cfg.DB)
	count, dbErr := queries.UpdateUserReading(
		r.Context(),
		database.UpdateUserReadingParams{
			UserID: userUUID,
			BookID: userReading.bookID,
			Status: userReading.status,
			Rating: userReading.rating, StartDate: userReading.startDate, FinishDate: userReading.finishDate})
	if count == 0 {
		common.RespondWithError(w, http.StatusBadRequest, "Unknown user reading")
		return
	}
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Delete user reading
// @Description Deletes user reading from DB. Uses access token from an HTTP-only cookie
// @Tags User reading
// @Accept json
// @Produce json
// @Param bookID path string true "Book ID"
// @Success 201 {string} string "Deleted successfully"
// @Failure 400 {object} ErrorResponse "Invalid bookID"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors/{bookID} [delete]
func (cfg *ApiConfig) HandleDeleteApiUserReadingPath(w http.ResponseWriter, r *http.Request) {
	bookID, err := uuid.Parse(r.PathValue("bookID"))
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid bookID")
		return
	}

	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	userID, usersStatusCode, err := clients.GetUser(r.Header, cfg.UsersServiceHost)
	if usersStatusCode == http.StatusUnauthorized {
		common.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to check authorization")
		return
	}

	queries := database.New(cfg.DB)
	dbErr := queries.DeleteUserReading(r.Context(), database.DeleteUserReadingParams{UserID: userID, BookID: bookID})
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, dbErr.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getBookIDs(userReading []ResponseUserReading) []string {
	res := make([]string, 0, len(userReading))
	for _, book := range userReading {
		res = append(res, book.ID)
	}
	return res
}

func getBookToResponseReading(userReading []ResponseUserReading) map[string]ResponseUserReading {
	res := make(map[string]ResponseUserReading)
	for _, book := range userReading {
		res[book.ID] = book
	}
	return res
}

func getUserReading(db *sql.DB, userID uuid.UUID, ctx context.Context) ([]ResponseUserReading, error) {
	queries := database.New(db)
	userReading, dbErr := queries.GetUserReading(ctx, userID)
	if dbErr != nil {
		return nil, dbErr
	}
	res := make([]ResponseUserReading, 0, len(userReading))
	for _, book := range userReading {
		res = append(res, ResponseUserReading{ID: book.BookID.String(), Status: string(book.Status), Rating: int(book.Rating)})
	}
	return res, nil
}

func getUserReadingByStatus(db *sql.DB, userID uuid.UUID, status database.ReadingStatus, ctx context.Context) ([]ResponseUserReading, error) {
	queries := database.New(db)
	userReading, dbErr := queries.GetUserReadingByStatus(ctx, database.GetUserReadingByStatusParams{UserID: userID, Status: status})
	if dbErr != nil {
		return nil, dbErr
	}
	res := make([]ResponseUserReading, 0, len(userReading))
	for _, book := range userReading {
		res = append(res, ResponseUserReading{ID: book.BookID.String(), Status: string(status), Rating: int(book.Rating)})
	}
	return res, nil
}

// @Summary Get user reading
// @Description Gets user reading from DB. Uses access token from an HTTP-only cookie
// @Tags User reading
// @Accept json
// @Produce json
// @Param status query string false "Reading status"
// @Success 200 {array} ResponseUserReading "User reading"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors [get]
func (cfg *ApiConfig) HandleGetApiUserReadingPath(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	userID, usersStatusCode, err := clients.GetUser(r.Header, cfg.UsersServiceHost)
	if usersStatusCode == http.StatusUnauthorized {
		common.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to check authorization")
		return
	}

	requestStatus := r.URL.Query().Get("status")
	userReading := []ResponseUserReading{}
	if requestStatus == "" {
		userReading, err = getUserReading(cfg.DB, userID, r.Context())
	} else {
		dbStatus, err := mapUserReadingStatus(requestStatus)
		if err != nil {
			common.RespondWithError(w, http.StatusBadRequest, "Unknown reading status")
			return
		}
		userReading, err = getUserReadingByStatus(cfg.DB, userID, dbStatus, r.Context())
	}
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(userReading) == 0 {
		common.RespondWithJSON(w, http.StatusOK, []ResponseUserReading{}, nil)
		return
	}

	bookIDs := getBookIDs(userReading)
	statusCode, booksInfo, err := clients.GetBooksInfo(bookIDs, cfg.LibraryServiceHost)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if statusCode != http.StatusOK {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to get books info")
		return
	}
	bookToResponseReading := getBookToResponseReading(userReading)
	response := []ResponseUserReading{}
	for _, bookInfo := range booksInfo {
		responseReading, ok := bookToResponseReading[bookInfo.ID]
		if !ok {
			continue
		}
		responseReading.Title = bookInfo.Title
		responseReading.Authors = bookInfo.Authors
		response = append(response, responseReading)
	}

	common.RespondWithJSON(w, http.StatusOK, response, nil)
}

// @Summary Get one user reading full info
// @Description Gets one user reading full info from DB. Uses access token from an HTTP-only cookie
// @Tags User reading
// @Accept json
// @Produce json
// @Param bookID path string true "Book ID"
// @Success 200 {object} ResponseUserReadingFullInfo "User reading full info"
// @Failure 401 {object} ErrorResponse "Unauthorized"
// @Failure 500 {object} ErrorResponse
// @Router /api/authors/{bookID} [get]
func (cfg *ApiConfig) HandleGetApiUserReadingByBookPath(w http.ResponseWriter, r *http.Request) {
	if cfg.DB == nil {
		common.RespondWithError(w, http.StatusInternalServerError, "DB error")
		return
	}

	userID, usersStatusCode, err := clients.GetUser(r.Header, cfg.UsersServiceHost)
	if usersStatusCode == http.StatusUnauthorized {
		common.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to check authorization")
		return
	}

	bookID, err := uuid.Parse(r.PathValue("bookID"))
	if err != nil {
		common.RespondWithError(w, http.StatusBadRequest, "Invalid book id")
		return
	}

	queries := database.New(cfg.DB)
	userReading, dbErr := queries.GetUserReadingByBook(r.Context(), database.GetUserReadingByBookParams{UserID: userID, BookID: bookID})
	if dbErr == sql.ErrNoRows {
		common.RespondWithError(w, http.StatusNotFound, "Unknown user book")
		return
	}
	if dbErr != nil {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to get user reading")
		return
	}

	statusCode, booksInfo, err := clients.GetBooksInfo([]string{bookID.String()}, cfg.LibraryServiceHost)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if statusCode != http.StatusOK {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to get books info")
		return
	}
	if len(booksInfo) == 0 {
		common.RespondWithError(w, http.StatusNotFound, "Unknown book")
		return
	}
	response := ResponseUserReadingFullInfo{
		ResponseUserReading: ResponseUserReading{
			ID:      bookID.String(),
			Title:   booksInfo[0].Title,
			Authors: booksInfo[0].Authors,
			Status:  string(userReading.Status),
			Rating:  int(userReading.Rating),
		},
		StartDate:  common.NullTimeToString(userReading.StartDate),
		FinishDate: common.NullTimeToString(userReading.FinishDate),
	}
	common.RespondWithJSON(w, http.StatusOK, response, nil)
}

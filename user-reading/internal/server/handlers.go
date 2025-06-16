package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/bakurvik/mylib/user-reading/internal/clients"
	"github.com/bakurvik/mylib/user-reading/internal/database"
	"github.com/google/uuid"

	common "github.com/bakurvik/mylib-common"
)

const (
	readingStatus    = database.ReadingStatusReading
	wantToReadStatus = database.ReadingStatusWantToRead
	finishedStatus   = database.ReadingStatusFinished
)

type dbUserReading struct {
	bookID     uuid.UUID
	status     database.ReadingStatus
	rating     int32
	startDate  sql.NullTime
	finishDate sql.NullTime
	createdAt  time.Time
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

func parseUserReading(r *http.Request) (dbUserReading, error) {
	decoder := json.NewDecoder(r.Body)
	request := UserReading{}
	err := decoder.Decode(&request)
	if err != nil {
		return dbUserReading{}, err
	}
	bookUUID, err := uuid.Parse(request.BookID)
	if err != nil {
		return dbUserReading{}, err
	}
	status, err := mapUserReadingStatus(request.Status)
	if err != nil {
		return dbUserReading{}, err
	}
	startDate := common.ToNullTime(request.StartDate)
	finishDate := common.ToNullTime(request.FinishDate)
	if startDate.Valid && finishDate.Valid && startDate.Time.After(finishDate.Time) {
		return dbUserReading{}, errors.New("Invalid start and finish dates")
	}
	res := dbUserReading{bookID: bookUUID, status: status, startDate: startDate, finishDate: finishDate}
	if status == database.ReadingStatusFinished {
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

func getBookIDs(userReading []dbUserReading) []string {
	res := make([]string, 0, len(userReading))
	for _, book := range userReading {
		res = append(res, book.bookID.String())
	}
	return res
}

func getUserReading(db *sql.DB, userID uuid.UUID, ctx context.Context) ([]dbUserReading, error) {
	queries := database.New(db)
	userReading, dbErr := queries.GetUserReading(ctx, userID)
	if dbErr != nil {
		return nil, dbErr
	}
	res := make([]dbUserReading, 0, len(userReading))
	for _, book := range userReading {
		res = append(res, dbUserReading{
			bookID:     book.BookID,
			status:     book.Status,
			rating:     book.Rating,
			startDate:  book.StartDate,
			finishDate: book.FinishDate,
			createdAt:  book.CreatedAt})
	}
	return res, nil
}

func getUserReadingByStatus(db *sql.DB, userID uuid.UUID, status database.ReadingStatus, ctx context.Context) ([]dbUserReading, error) {
	queries := database.New(db)
	userReading, dbErr := queries.GetUserReadingByStatus(ctx, database.GetUserReadingByStatusParams{UserID: userID, Status: status})
	if dbErr != nil {
		return nil, dbErr
	}
	res := make([]dbUserReading, 0, len(userReading))
	for _, book := range userReading {
		res = append(res, dbUserReading{
			bookID:     book.BookID,
			status:     status,
			rating:     book.Rating,
			startDate:  book.StartDate,
			finishDate: book.FinishDate,
			createdAt:  book.CreatedAt})
	}
	return res, nil
}

func compareDates(left dbUserReading, right dbUserReading, getDateField func(dbUserReading) sql.NullTime) bool {
	leftDate := getDateField(left)
	rightDate := getDateField(right)
	if !leftDate.Valid && !rightDate.Valid {
		return left.createdAt.After(right.createdAt)
	}
	if !leftDate.Valid {
		return true
	}
	if !rightDate.Valid {
		return false
	}
	return leftDate.Time.After(rightDate.Time)
}

func sortUserReading(books []dbUserReading) {
	priority := map[database.ReadingStatus]int{
		readingStatus:    0,
		wantToReadStatus: 1,
		finishedStatus:   2,
	}

	sort.Slice(books, func(i, j int) bool {
		if books[i].status != books[j].status {
			return priority[books[i].status] < priority[books[j].status]
		}
		if books[i].status == readingStatus {
			return compareDates(books[i], books[j], func(a dbUserReading) sql.NullTime { return a.startDate })
		}
		if books[i].status == finishedStatus {
			return compareDates(books[i], books[j], func(a dbUserReading) sql.NullTime { return a.finishDate })
		}
		return books[i].createdAt.After(books[j].createdAt)
	})
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
	userReading := []dbUserReading{}
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
	statusCode, idToBookInfo, err := clients.GetBooksInfo(bookIDs, cfg.LibraryServiceHost, cfg.BooksCacheCfg)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if statusCode != http.StatusOK {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to get books info")
		return
	}
	sortUserReading(userReading)
	response := []ResponseUserReading{}
	for _, userReading := range userReading {
		bookInfo, ok := idToBookInfo[userReading.bookID.String()]
		if !ok {
			continue
		}
		respUserReading := ResponseUserReading{
			ID:      userReading.bookID.String(),
			Title:   bookInfo.Title,
			Authors: bookInfo.Authors,
			Status:  string(userReading.status),
			Rating:  int(userReading.rating),
		}
		response = append(response, respUserReading)
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

	statusCode, booksInfo, err := clients.GetBooksInfo([]string{bookID.String()}, cfg.LibraryServiceHost, cfg.BooksCacheCfg)
	if err != nil {
		common.RespondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if statusCode != http.StatusOK {
		common.RespondWithError(w, http.StatusInternalServerError, "Failed to get books info")
		return
	}
	bookInfo, ok := booksInfo[bookID.String()]
	if !ok {
		common.RespondWithError(w, http.StatusNotFound, "Unknown book")
		return
	}
	response := ResponseUserReadingFullInfo{
		ResponseUserReading: ResponseUserReading{
			ID:      bookID.String(),
			Title:   bookInfo.Title,
			Authors: bookInfo.Authors,
			Status:  string(userReading.Status),
			Rating:  int(userReading.Rating),
		},
		StartDate:  common.NullTimeToString(userReading.StartDate),
		FinishDate: common.NullTimeToString(userReading.FinishDate),
	}
	common.RespondWithJSON(w, http.StatusOK, response, nil)
}

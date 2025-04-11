package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
	"users/internal/auth"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	apiUsersPath = "/api/users"
	selectUsers  = "SELECT login_name, email, birth_date, hashed_password, created_at, updated_at FROM users WHERE id = $1"
)

type ResponseUser struct {
	Login     string `json:"login"`
	Email     string `json:"email,omitempty"`
	BirthDate string `json:"birth_date,omitempty"`
}

func getDbUser(db *sql.DB, id string) *User {
	row := db.QueryRow(selectUsers, id)
	user := User{}
	err := row.Scan(&user.loginName, &user.email, &user.birthDate, &user.hashedPassword, &user.createdAt, &user.updatedAt)
	if err != nil {
		return nil
	}
	return &user
}

func getUserFromResponse(response *http.Response) ResponseUser {
	body, _ := io.ReadAll(response.Body)
	responseData := ResponseUser{}
	json.Unmarshal(body, &responseData)
	return responseData
}

func TestCreateUser_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		LoginName string `json:"login_name"`
		Email     string `json:"email"`
		BirthDate string `json:"birth_date,omitempty"`
		Password  string `json:"password"`
	}
	request := requestBody{LoginName: "login", Email: "login@email.ru", BirthDate: "01.02.2003", Password: "password"}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(server.URL+apiUsersPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusCreated, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	type responseUser struct {
		ID    string `json:"id"`
		Token string `json:"token"`
	}
	responseBody := responseUser{}
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.NotEqual(t, responseBody.Token, "")

	cookies := response.Cookies()
	assert.Equal(t, len(cookies), 1)
	cookie := cookies[0]
	assert.Equal(t, cookie.Name, cookieRefreshToken)
	assert.NotEqual(t, cookie.Value, "")

	user := getDbUser(db, responseBody.ID)

	assert.Equal(t, user.loginName, request.LoginName)
	assert.Equal(t, user.email, request.Email)
	assert.Equal(t, user.birthDate.Time.Format(timeFormat), request.BirthDate)
	assert.NotEqual(t, user.hashedPassword, request.Password)

	newRefreshToken := getDbToken(db, cookie.Value)
	assert.Equal(t, newRefreshToken.userId, responseBody.ID)
	assert.False(t, newRefreshToken.revokedAt.Valid)
}

func TestCreateUser_LoginExists(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	login_name := "some_login"
	addDBUser(db, User{loginName: login_name, email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"})

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		LoginName string `json:"login_name"`
		Email     string `json:"email"`
		BirthDate string `json:"birth_date,omitempty"`
		Password  string `json:"password"`
	}
	request := requestBody{LoginName: login_name, Email: "another_login@email.ru", BirthDate: "01.02.2003", Password: "password"}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(server.URL+apiUsersPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusConflict, response.StatusCode)
}

func TestCreateUser_EmailExists(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	email := "some_email@email.com"
	addDBUser(db, User{loginName: "some_login", email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"})

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		LoginName string `json:"login_name"`
		Email     string `json:"email"`
		BirthDate string `json:"birth_date,omitempty"`
		Password  string `json:"password"`
	}
	request := requestBody{LoginName: "another_login", Email: email, BirthDate: "01.02.2003", Password: "password"}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(server.URL+apiUsersPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusConflict, response.StatusCode)
}

func TestUpdateUser_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	userID := addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		LoginName string `json:"login_name"`
		Email     string `json:"email"`
		BirthDate string `json:"birth_date,omitempty"`
		Password  string `json:"password"`
	}
	newPassword := "another_password"
	req := requestBody{LoginName: "another_login", Email: "another_login@email.ru", BirthDate: "10.05.2014", Password: newPassword}
	requestJson, _ := json.Marshal(req)

	request, requestErr := http.NewRequest("PUT", server.URL+apiUsersPath, bytes.NewBuffer(requestJson))
	assert.NoError(t, requestErr)
	uuid, _ := uuid.Parse(userID)
	accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
	request.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNoContent, response.StatusCode)

	user := getDbUser(db, userID)

	assert.Equal(t, user.loginName, req.LoginName)
	assert.Equal(t, user.email, req.Email)
	assert.Equal(t, user.birthDate.Time.Format(timeFormat), req.BirthDate)
	assert.Equal(t, auth.CheckPasswordHash(user.hashedPassword, newPassword), nil)
}

func TestUpdateUser_InvalidToken(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		LoginName string `json:"login_name"`
		Email     string `json:"email"`
		BirthDate string `json:"birth_date,omitempty"`
		Password  string `json:"password"`
	}
	newPassword := "another_password"
	req := requestBody{LoginName: "another_login", Email: "another_login@email.ru", BirthDate: "10.05.2014", Password: newPassword}
	requestJson, _ := json.Marshal(req)

	request, requestErr := http.NewRequest("PUT", server.URL+apiUsersPath, bytes.NewBuffer(requestJson))
	assert.NoError(t, requestErr)
	request.Header.Add("Authorization", "Bearer invalid_token")

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestUpdateUser_NoToken(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		LoginName string `json:"login_name"`
		Email     string `json:"email"`
		BirthDate string `json:"birth_date,omitempty"`
		Password  string `json:"password"`
	}
	newPassword := "another_password"
	req := requestBody{LoginName: "another_login", Email: "another_login@email.ru", BirthDate: "10.05.2014", Password: newPassword}
	requestJson, _ := json.Marshal(req)

	request, requestErr := http.NewRequest("PUT", server.URL+apiUsersPath, bytes.NewBuffer(requestJson))
	assert.NoError(t, requestErr)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestGetUser_AuthorizedAsRequestUser(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	user := User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"}
	userID := addDBUser(db, user)

	server := setupTestServer(db)
	defer server.Close()

	request, requestErr := http.NewRequest("GET", fmt.Sprintf("%v%v/{%v}", server.URL, apiUsersPath, userID), nil)
	assert.NoError(t, requestErr)
	uuid, _ := uuid.Parse(userID)
	accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
	request.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	responseUser := getUserFromResponse(response)
	assert.Equal(t, responseUser.Login, user.loginName)
	assert.Equal(t, responseUser.Email, user.email)
	assert.Equal(t, responseUser.BirthDate, user.birthDate.Time.Format(timeFormat))
}

func TestGetUser_AuthorizedAsAnotherUser(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	user := User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"}
	userID := addDBUser(db, user)

	server := setupTestServer(db)
	defer server.Close()

	request, requestErr := http.NewRequest("GET", fmt.Sprintf("%v%v/{%v}", server.URL, apiUsersPath, userID), nil)
	assert.NoError(t, requestErr)
	anotherUserId := "4fc40366-ff15-4653-be30-1bba21f016c1"
	uuid, _ := uuid.Parse(anotherUserId)
	accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
	request.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	responseUser := getUserFromResponse(response)
	assert.Equal(t, responseUser.Login, user.loginName)
	assert.Equal(t, responseUser.Email, "")
	assert.Equal(t, responseUser.BirthDate, "")
}

func TestGetUser_NonAuthorized(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	user := User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"}
	userID := addDBUser(db, user)

	server := setupTestServer(db)
	defer server.Close()

	request, requestErr := http.NewRequest("GET", fmt.Sprintf("%v%v/{%v}", server.URL, apiUsersPath, userID), nil)
	assert.NoError(t, requestErr)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	responseUser := getUserFromResponse(response)
	assert.Equal(t, responseUser.Login, user.loginName)
	assert.Equal(t, responseUser.Email, "")
	assert.Equal(t, responseUser.BirthDate, "")
}

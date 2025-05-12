package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	common "github.com/bakurvik/mylib-common"
	"github.com/bakurvik/mylib/users/internal/auth"
	"github.com/bakurvik/mylib/users/internal/server"
	"github.com/google/uuid"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	insertRefreshToken = "INSERT INTO refresh_tokens(token, user_id, expires_at, revoked_at) VALUES (gen_random_uuid(), $1, $2, $3) RETURNING token"
)

func addDBToken(db *sql.DB, refreshToken RefreshToken) string {
	row := db.QueryRow(
		insertRefreshToken, refreshToken.userId, refreshToken.expiresAt, refreshToken.revokedAt)
	token := ""
	err := row.Scan(&token)
	if err != nil {
		log.Print("DB error: ", err)
	}
	return token
}

func TestLogin_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	email := "some_email@email.com"
	password := "some_password"
	hash, _ := auth.HashPassword(password)
	addDBUser(db, User{loginName: "login", email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: hash})

	s := setupTestServer(db)
	defer s.Close()

	request := server.RequestLogin{Password: password, Email: email}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(s.URL+server.AuthLoginPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := server.ResponseToken{}
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.NotEqual(t, responseBody.Token, "")

	cookies := response.Cookies()
	assert.Equal(t, len(cookies), 1)
	cookie := cookies[0]
	assert.Equal(t, cookie.Name, "refresh_token")
	assert.NotEqual(t, cookie.Value, "")

	newRefreshToken := getDbToken(db, cookie.Value)
	assert.Equal(t, newRefreshToken.userId, responseBody.ID)
	assert.False(t, newRefreshToken.revokedAt.Valid)
}

func TestLogin_InvalidPassword(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	email := "some_email@email.com"
	password := "some_password"
	login := "login"
	hash, _ := auth.HashPassword(password)
	addDBUser(db, User{loginName: login, email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: hash})

	s := setupTestServer(db)
	defer s.Close()

	anotherPassword := "another_password"
	request := server.RequestUser{LoginName: login, Password: anotherPassword, Email: email}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(s.URL+server.AuthLoginPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestLogin_NoUser(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	email := "some_email@email.com"
	password := "some_password"
	request := server.RequestUser{LoginName: "login", Password: password, Email: email}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(s.URL+server.AuthLoginPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

func TestRefresh_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	userID := addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})
	expiresAt := time.Now().Add(time.Hour)
	token := addDBToken(db, RefreshToken{userId: userID, expiresAt: expiresAt})

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("POST", s.URL+server.AuthRefreshPath, nil)
	assert.NoError(t, requestErr)
	request.AddCookie(&http.Cookie{
		Name:     cookieRefreshToken,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	decoder := json.NewDecoder(response.Body)
	responseBody := server.ResponseToken{}
	err = decoder.Decode(&responseBody)
	assert.NoError(t, err)
	assert.NotEqual(t, responseBody.Token, "")

	cookies := response.Cookies()
	assert.Equal(t, len(cookies), 1)
	cookie := cookies[0]
	assert.Equal(t, cookie.Name, "refresh_token")
	assert.NotEqual(t, cookie.Value, "")
	assert.NotEqual(t, cookie.Value, token)

	newRefreshToken := getDbToken(db, cookie.Value)
	assert.Equal(t, newRefreshToken.userId, userID)
	assert.False(t, newRefreshToken.revokedAt.Valid)

	oldRefreshToken := getDbToken(db, token)
	assert.True(t, oldRefreshToken.revokedAt.Valid)
}

func TestRefresh_NoCookie(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	userID := addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})
	expiresAt := time.Now().Add(time.Hour)
	addDBToken(db, RefreshToken{userId: userID, expiresAt: expiresAt})

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("POST", s.URL+server.AuthRefreshPath, nil)
	assert.NoError(t, requestErr)

	client := &http.Client{}
	response, responseErr := client.Do(request)
	assert.NoError(t, responseErr)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestRefresh_UnknownToken(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("POST", s.URL+server.AuthRefreshPath, nil)
	assert.NoError(t, requestErr)
	request.AddCookie(&http.Cookie{
		Name:     cookieRefreshToken,
		Value:    "4500f6128a7209ebdc18de559daf74f5",
		Path:     "/",
		Expires:  time.Now().Add(time.Hour),
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestRevoke_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	userID := addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})
	expiresAt := time.Now().Add(time.Hour)
	token := addDBToken(db, RefreshToken{userId: userID, expiresAt: expiresAt})

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("POST", s.URL+server.AuthRevokePath, nil)
	assert.NoError(t, requestErr)
	request.AddCookie(&http.Cookie{
		Name:     cookieRefreshToken,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNoContent, response.StatusCode)

	oldRefreshToken := getDbToken(db, token)
	assert.True(t, oldRefreshToken.revokedAt.Valid)
}

func TestRevoke_NoCookie(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	userID := addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})
	expiresAt := time.Now().Add(time.Hour)
	token := addDBToken(db, RefreshToken{userId: userID, expiresAt: expiresAt})

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("POST", s.URL+server.AuthRevokePath, nil)
	assert.NoError(t, requestErr)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)

	oldRefreshToken := getDbToken(db, token)
	assert.False(t, oldRefreshToken.revokedAt.Valid)
}

func TestRevoke_UnknownToken(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("POST", s.URL+server.AuthRevokePath, nil)
	assert.NoError(t, requestErr)
	request.AddCookie(&http.Cookie{
		Name:     cookieRefreshToken,
		Value:    "4500f6128a7209ebdc18de559daf74f5",
		Path:     "/",
		Expires:  time.Now().Add(time.Hour),
		Secure:   false,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusNoContent, response.StatusCode)
}

func TestPing_Success(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.PingPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestWhoami_Authorized(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)
	cleanupDB(db)
	user := User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"}
	userID := addDBUser(db, user)

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("GET", s.URL+server.AuthWhoamiPath, nil)
	assert.NoError(t, requestErr)
	uuid, _ := uuid.Parse(userID)
	accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
	request.Header.Add("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)

	body, _ := io.ReadAll(response.Body)
	responseData := server.ResponseUserID{}
	json.Unmarshal(body, &responseData)
	assert.Equal(t, responseData.ID, userID)
}

func TestWhoami_Unauthorized(t *testing.T) {
	db, err := common.SetupDBByUrl("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)

	s := setupTestServer(db)
	defer s.Close()

	request, requestErr := http.NewRequest("GET", s.URL+server.AuthWhoamiPath, nil)
	assert.NoError(t, requestErr)

	client := &http.Client{}
	response, err := client.Do(request)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

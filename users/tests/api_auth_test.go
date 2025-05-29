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
		insertRefreshToken, refreshToken.userID, refreshToken.expiresAt, refreshToken.revokedAt)
	token := ""
	err := row.Scan(&token)
	if err != nil {
		log.Print("DB error: ", err)
	}
	return token
}

func TestLogin(t *testing.T) {
	email := "some_email@email.com"
	password := "some_password"
	type testCase struct {
		name               string
		request            server.RequestLogin
		expectedStatusCode int
	}
	testCases := []testCase{
		{
			name:               "success",
			request:            server.RequestLogin{Password: password, Email: email},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "invalid_password",
			request:            server.RequestLogin{Password: "invalid_password", Email: email},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:               "unknown_user",
			request:            server.RequestLogin{Password: password, Email: "unknown_email@email.com"},
			expectedStatusCode: http.StatusNotFound,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			hash, _ := auth.HashPassword(password)
			addDBUser(db, User{loginName: "login", email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: hash})

			s := setupTestServer(db)
			defer s.Close()

			requestJson, _ := json.Marshal(tc.request)

			response, err := http.Post(s.URL+server.AuthLoginPath, "application/json", bytes.NewBuffer(requestJson))
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
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

				newRefreshToken := getDBToken(db, cookie.Value)
				assert.Equal(t, newRefreshToken.userID, responseBody.ID)
				assert.False(t, newRefreshToken.revokedAt.Valid)
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	type testCase struct {
		name               string
		expectedStatusCode int
		hasCookie          bool
		cookieToken        string
	}
	testCases := []testCase{
		{
			name:               "success",
			expectedStatusCode: http.StatusOK,
			hasCookie:          true,
		},
		{
			name:               "no_cookie",
			expectedStatusCode: http.StatusUnauthorized,
			hasCookie:          false,
		},
		{
			name:               "unknown_token",
			expectedStatusCode: http.StatusUnauthorized,
			hasCookie:          false,
			cookieToken:        "4500f6128a7209ebdc18de559daf74f5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			userID := addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})
			expiresAt := time.Now().Add(time.Hour)
			token := addDBToken(db, RefreshToken{userID: userID, expiresAt: expiresAt})
			if tc.cookieToken != "" {
				token = tc.cookieToken
			}

			s := setupTestServer(db)
			defer s.Close()

			request, requestErr := http.NewRequest(http.MethodPost, s.URL+server.AuthRefreshPath, nil)
			assert.NoError(t, requestErr)
			if tc.hasCookie {
				request.AddCookie(&http.Cookie{
					Name:     cookieRefreshToken,
					Value:    token,
					Path:     "/",
					Expires:  expiresAt,
					Secure:   false,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			client := &http.Client{}
			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
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

				newRefreshToken := getDBToken(db, cookie.Value)
				assert.Equal(t, newRefreshToken.userID, userID)
				assert.False(t, newRefreshToken.revokedAt.Valid)

				oldRefreshToken := getDBToken(db, token)
				assert.True(t, oldRefreshToken.revokedAt.Valid)
			}
		})
	}
}

func TestRevoke(t *testing.T) {
	type testCase struct {
		name                   string
		hasCookie              bool
		cookieToken            string
		expectedStatusCode     int
		expectedRevokedAtValid bool
	}
	testCases := []testCase{
		{
			name:                   "success",
			hasCookie:              true,
			expectedStatusCode:     http.StatusNoContent,
			expectedRevokedAtValid: true,
		},
		{
			name:                   "no_cookie",
			hasCookie:              false,
			expectedStatusCode:     http.StatusUnauthorized,
			expectedRevokedAtValid: false,
		},
		{
			name:                   "unknown_token",
			hasCookie:              true,
			cookieToken:            "4500f6128a7209ebdc18de559daf74f5",
			expectedStatusCode:     http.StatusNoContent,
			expectedRevokedAtValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			userID := addDBUser(db, User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"})
			expiresAt := time.Now().Add(time.Hour)
			token := addDBToken(db, RefreshToken{userID: userID, expiresAt: expiresAt})
			if tc.cookieToken != "" {
				token = tc.cookieToken
			}

			s := setupTestServer(db)
			defer s.Close()

			request, requestErr := http.NewRequest(http.MethodPost, s.URL+server.AuthRevokePath, nil)
			assert.NoError(t, requestErr)
			if tc.hasCookie {
				request.AddCookie(&http.Cookie{
					Name:     cookieRefreshToken,
					Value:    token,
					Path:     "/",
					Expires:  expiresAt,
					Secure:   false,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
			}

			client := &http.Client{}
			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if tc.cookieToken == "" {
				oldRefreshToken := getDBToken(db, token)
				assert.Equal(t, oldRefreshToken.revokedAt.Valid, tc.expectedRevokedAtValid)
			}
		})
	}
}

func TestPing_Success(t *testing.T) {
	db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
	assert.NoError(t, err)
	defer common.CloseDB(db)

	s := setupTestServer(db)
	defer s.Close()

	response, err := http.Get(s.URL + server.PingPath)
	assert.NoError(t, err)
	defer common.CloseResponseBody(response)
	assert.Equal(t, http.StatusOK, response.StatusCode)
}

func TestWhoami(t *testing.T) {
	type testCase struct {
		name               string
		hasToken           bool
		expectedStatusCode int
	}
	testCases := []testCase{
		{
			name:               "success",
			hasToken:           true,
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "unauthorized",
			hasToken:           false,
			expectedStatusCode: http.StatusUnauthorized,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			user := User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "304854e2e79de0f96dc5477fef38a18f"}
			userID := addDBUser(db, user)

			s := setupTestServer(db)
			defer s.Close()

			request, requestErr := http.NewRequest(http.MethodGet, s.URL+server.AuthWhoamiPath, nil)
			assert.NoError(t, requestErr)
			if tc.hasToken {
				uuid, _ := uuid.Parse(userID)
				accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
				request.Header.Add("Authorization", "Bearer "+accessToken)
			}

			client := &http.Client{}
			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if response.StatusCode == http.StatusOK {
				body, _ := io.ReadAll(response.Body)
				responseData := server.ResponseUserID{}
				err := json.Unmarshal(body, &responseData)
				assert.NoError(t, err)
				assert.Equal(t, responseData.ID, userID)
			}
		})
	}
}

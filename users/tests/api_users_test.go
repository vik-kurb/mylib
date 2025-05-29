package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
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
	selectUsers = "SELECT login_name, email, birth_date, hashed_password FROM users WHERE id = $1"
)

func getDBUser(db *sql.DB, id string) *User {
	row := db.QueryRow(selectUsers, id)
	user := User{}
	err := row.Scan(&user.loginName, &user.email, &user.birthDate, &user.hashedPassword)
	if err != nil {
		return nil
	}
	if user.birthDate.Valid {
		user.birthDate.Time = user.birthDate.Time.In(time.UTC)
	}
	return &user
}

func getUserFromResponse(response *http.Response) server.ResponseUser {
	body, _ := io.ReadAll(response.Body)
	responseData := server.ResponseUser{}
	err := json.Unmarshal(body, &responseData)
	if err != nil {
		log.Print("Failed to get user from response: ", err)
	}
	return responseData
}

func TestCreateUser(t *testing.T) {
	email := "login@email.com"
	password := "some_password"
	login := "login"
	type testCase struct {
		name               string
		dbUser             User
		request            server.RequestUser
		expectedStatusCode int
	}
	testCases := []testCase{
		{
			name:               "success",
			request:            server.RequestUser{LoginName: login, Email: email, BirthDate: "01.02.2003", Password: password},
			expectedStatusCode: http.StatusCreated,
		},
		{
			name:               "login_exists",
			dbUser:             User{loginName: login, email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"},
			request:            server.RequestUser{LoginName: login, Email: email, BirthDate: "01.02.2003", Password: password},
			expectedStatusCode: http.StatusConflict,
		},
		{
			name:               "email_exists",
			dbUser:             User{loginName: "new_login", email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"},
			request:            server.RequestUser{LoginName: login, Email: email, BirthDate: "01.02.2003", Password: password},
			expectedStatusCode: http.StatusConflict,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			addDBUser(db, tc.dbUser)

			s := setupTestServer(db)
			defer s.Close()

			requestJson, _ := json.Marshal(tc.request)

			response, err := http.Post(s.URL+server.ApiUsersPath, "application/json", bytes.NewBuffer(requestJson))
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			if response.StatusCode == http.StatusCreated {
				decoder := json.NewDecoder(response.Body)
				responseBody := server.ResponseToken{}
				err = decoder.Decode(&responseBody)
				assert.NoError(t, err)
				assert.NotEqual(t, responseBody.Token, "")

				cookies := response.Cookies()
				assert.Equal(t, len(cookies), 1)
				cookie := cookies[0]
				assert.Equal(t, cookie.Name, cookieRefreshToken)
				assert.NotEqual(t, cookie.Value, "")

				user := getDBUser(db, responseBody.ID)

				assert.Equal(t, user.loginName, tc.request.LoginName)
				assert.Equal(t, user.email, tc.request.Email)
				assert.Equal(t, user.birthDate.Time.Format(timeFormat), tc.request.BirthDate)
				assert.NotEqual(t, user.hashedPassword, tc.request.Password)

				newRefreshToken := getDBToken(db, cookie.Value)
				assert.Equal(t, newRefreshToken.userID, responseBody.ID)
				assert.False(t, newRefreshToken.revokedAt.Valid)
			}
		})
	}
}

func TestUpdateUser(t *testing.T) {
	newPassword := "another_password"
	type testCase struct {
		name               string
		dbUser             User
		request            server.RequestUser
		requestToken       string
		expectedStatusCode int
	}
	testCases := []testCase{
		{
			name:               "success",
			dbUser:             User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"},
			request:            server.RequestUser{LoginName: "another_login", Email: "another_login@email.ru", BirthDate: "10.05.2014", Password: newPassword},
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "invalid_token",
			dbUser:             User{loginName: "login", email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"},
			request:            server.RequestUser{LoginName: "another_login", Email: "another_login@email.ru", BirthDate: "10.05.2014", Password: newPassword},
			requestToken:       "invalid_token",
			expectedStatusCode: http.StatusUnauthorized,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := common.SetupDBByURL("../.env", "TEST_DB_URL")
			assert.NoError(t, err)
			defer common.CloseDB(db)
			cleanupDB(db)
			userID := addDBUser(db, tc.dbUser)

			s := setupTestServer(db)
			defer s.Close()

			requestJson, _ := json.Marshal(tc.request)

			request, requestErr := http.NewRequest(http.MethodPut, s.URL+server.ApiUsersPath, bytes.NewBuffer(requestJson))
			assert.NoError(t, requestErr)
			uuid, _ := uuid.Parse(userID)
			accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
			if tc.requestToken != "" {
				accessToken = tc.requestToken
			}
			request.Header.Add("Authorization", "Bearer "+accessToken)

			client := &http.Client{}
			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			user := getDBUser(db, userID)

			if response.StatusCode == http.StatusNoContent {
				assert.Equal(t, user.loginName, tc.request.LoginName)
				assert.Equal(t, user.email, tc.request.Email)
				assert.Equal(t, user.birthDate.Time.Format(timeFormat), tc.request.BirthDate)
				assert.Nil(t, auth.CheckPasswordHash(user.hashedPassword, newPassword))
			} else {
				assert.Equal(t, user.loginName, tc.dbUser.loginName)
				assert.Equal(t, user.email, tc.dbUser.email)
				assert.Equal(t, user.birthDate, tc.dbUser.birthDate)
				assert.Equal(t, user.hashedPassword, tc.dbUser.hashedPassword)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	anotherUserID := "4fc40366-ff15-4653-be30-1bba21f016c1"
	anotherUseruuid, _ := uuid.Parse(anotherUserID)
	accessToken, _ := auth.MakeJWT(anotherUseruuid, authSecretKey, time.Hour)
	type testCase struct {
		name               string
		token              string
		expectedStatusCode int
		expectedUser       server.ResponseUser
	}
	testCases := []testCase{
		{
			name:               "authorized_as_request_user",
			token:              "",
			expectedStatusCode: http.StatusOK,
			expectedUser:       server.ResponseUser{LoginName: "login", Email: "some_email@email.com", BirthDate: "09.05.1956"},
		},
		{
			name:               "authorized_as_another_user",
			token:              accessToken,
			expectedStatusCode: http.StatusOK,
			expectedUser:       server.ResponseUser{LoginName: "login"},
		},
		{
			name:               "unauthorized",
			token:              "invalid_token",
			expectedStatusCode: http.StatusOK,
			expectedUser:       server.ResponseUser{LoginName: "login"},
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

			request, requestErr := http.NewRequest(http.MethodGet, fmt.Sprintf("%v%v/{%v}", s.URL, server.ApiUsersPath, userID), nil)
			assert.NoError(t, requestErr)
			uuid, _ := uuid.Parse(userID)
			accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
			if tc.token != "" {
				accessToken = tc.token
			}
			request.Header.Add("Authorization", "Bearer "+accessToken)

			client := &http.Client{}
			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			responseUser := getUserFromResponse(response)
			assert.Equal(t, responseUser, tc.expectedUser)
		})
	}
}

func TestDeleteUser(t *testing.T) {
	type testCase struct {
		name               string
		hasToken           bool
		expectedStatusCode int
	}
	testCases := []testCase{
		{
			name:               "success",
			hasToken:           true,
			expectedStatusCode: http.StatusNoContent,
		},
		{
			name:               "success",
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

			request, requestErr := http.NewRequest(http.MethodDelete, s.URL+server.ApiUsersPath, nil)
			assert.NoError(t, requestErr)
			uuid, _ := uuid.Parse(userID)
			accessToken, _ := auth.MakeJWT(uuid, authSecretKey, time.Hour)
			if tc.hasToken {
				request.Header.Add("Authorization", "Bearer "+accessToken)
			}

			client := &http.Client{}
			response, err := client.Do(request)
			assert.NoError(t, err)
			defer common.CloseResponseBody(response)
			assert.Equal(t, tc.expectedStatusCode, response.StatusCode)

			dbUser := getDBUser(db, uuid.String())
			if tc.expectedStatusCode == http.StatusNoContent {
				assert.Nil(t, dbUser)
			} else {
				assert.NotNil(t, dbUser)
				assert.Equal(t, *dbUser, user)
			}
		})
	}
}

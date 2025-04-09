package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
	"users/internal/auth"
	"users/internal/database"
	"users/internal/server"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

const (
	apiUsersPath = "/api/users"
	apiLoginPath = "/api/login"
	selectUsers  = "SELECT login_name, email, birth_date, hashed_password, created_at, updated_at FROM users WHERE id = $1"
	deleteUsers  = "DELETE FROM users"
	timeFormat   = "02.01.2006"
)

type User struct {
	id             string
	loginName      string
	email          string
	birthDate      sql.NullTime
	hashedPassword string
	createdAt      time.Time
	updatedAt      time.Time
}

func setupTestDB() (*sql.DB, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		return nil, err
	}
	dbUrl := os.Getenv("TEST_DB_URL")
	return sql.Open("postgres", dbUrl)
}

func cleanupDB(db *sql.DB) {
	db.Query(deleteUsers)
}

func setupTestServer(db *sql.DB) *httptest.Server {
	apiCfg := server.ApiConfig{DB: database.New(db)}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm)
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

func addDbUser(db *sql.DB, user User) {
	db.Exec(
		"INSERT INTO users(id, login_name, email, birth_date, hashed_password) VALUES (gen_random_uuid(), $1, $2, $3, $4)",
		user.loginName, user.email, user.birthDate, user.hashedPassword)
}

func toSqlNullTime(s string) sql.NullTime {
	t, err := time.Parse(timeFormat, s)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
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
	assert.Equal(t, cookie.Name, "refresh_token")
	assert.NotEqual(t, cookie.Value, "")

	user := getDbUser(db, responseBody.ID)

	assert.Equal(t, user.loginName, request.LoginName)
	assert.Equal(t, user.email, request.Email)
	assert.Equal(t, user.birthDate.Time.Format(timeFormat), request.BirthDate)
	assert.NotEqual(t, user.hashedPassword, request.Password)
}

func TestCreateUser_LoginExists(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	login_name := "some_login"
	addDbUser(db, User{loginName: login_name, email: "some_email@email.com", birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"})

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
	addDbUser(db, User{loginName: "some_login", email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: "e51abab383822821e70b5b538901fbf7"})

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

func TestLogin_Success(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	email := "some_email@email.com"
	password := "some_password"
	hash, _ := auth.HashPassword(password)
	addDbUser(db, User{loginName: "login", email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: hash})

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	request := requestBody{Password: password, Email: email}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(server.URL+apiLoginPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

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
	assert.Equal(t, cookie.Name, "refresh_token")
	assert.NotEqual(t, cookie.Value, "")
}

func TestLogin_InvalidPassword(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)
	email := "some_email@email.com"
	password := "some_password"
	hash, _ := auth.HashPassword(password)
	addDbUser(db, User{loginName: "login", email: email, birthDate: toSqlNullTime("09.05.1956"), hashedPassword: hash})

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	anotherPassword := "another_password"
	request := requestBody{Password: anotherPassword, Email: email}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(server.URL+apiLoginPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

func TestLogin_NoUser(t *testing.T) {
	db, err := setupTestDB()
	assert.NoError(t, err)
	defer db.Close()
	cleanupDB(db)

	server := setupTestServer(db)
	defer server.Close()

	type requestBody struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	email := "some_email@email.com"
	password := "some_password"
	request := requestBody{Password: password, Email: email}
	requestJson, _ := json.Marshal(request)

	response, err := http.Post(server.URL+apiLoginPath, "application/json", bytes.NewBuffer(requestJson))
	assert.NoError(t, err)
	defer response.Body.Close()
	assert.Equal(t, http.StatusNotFound, response.StatusCode)
}

package tests

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"time"
	"users/internal/database"
	"users/internal/server"

	"github.com/joho/godotenv"
)

const (
	cookieRefreshToken = "refresh_token"
	insertUser         = "INSERT INTO users(id, login_name, email, birth_date, hashed_password) VALUES (gen_random_uuid(), $1, $2, $3, $4) RETURNING id"
	selectRefreshToken = "SELECT user_id, expires_at, revoked_at FROM refresh_tokens WHERE token = $1"
	deleteUsers        = "DELETE FROM users"
	authSecretKey      = "secret_key"
	timeFormat         = "02.01.2006"
)

type User struct {
	loginName      string
	email          string
	birthDate      sql.NullTime
	hashedPassword string
	createdAt      time.Time
	updatedAt      time.Time
}

type RefreshToken struct {
	userId    string
	expiresAt time.Time
	revokedAt sql.NullTime
}

func addDBUser(db *sql.DB, user User) string {
	row := db.QueryRow(
		insertUser,
		user.loginName, user.email, user.birthDate, user.hashedPassword)
	userID := ""
	row.Scan(&userID)
	return userID
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
	apiCfg := server.ApiConfig{DB: database.New(db), AuthSecretKey: authSecretKey}
	sm := http.NewServeMux()
	server.Handle(sm, &apiCfg)
	return httptest.NewServer(sm)
}

func getDbToken(db *sql.DB, token string) *RefreshToken {
	row := db.QueryRow(selectRefreshToken, token)
	dbToken := RefreshToken{}
	err := row.Scan(&dbToken.userId, &dbToken.expiresAt, &dbToken.revokedAt)
	if err != nil {
		return nil
	}
	return &dbToken
}

func toSqlNullTime(s string) sql.NullTime {
	t, err := time.Parse(timeFormat, s)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: t, Valid: true}
}

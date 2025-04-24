package tests

import (
	"database/sql"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/bakurvik/mylib/users/internal/database"
	"github.com/bakurvik/mylib/users/internal/server"
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
	err := row.Scan(&userID)
	if err != nil {
		log.Print("Failed to add user: ", err)
	}
	return userID
}

func cleanupDB(db *sql.DB) {
	_, err := db.Query(deleteUsers)
	if err != nil {
		log.Print("Failed to cleanup db: ", err)
	}
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

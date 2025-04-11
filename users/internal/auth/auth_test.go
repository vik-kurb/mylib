package auth

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHashPassword_Success(t *testing.T) {
	password := "password"
	hash, _ := HashPassword(password)
	err := CheckPasswordHash(hash, password)
	assert.NoError(t, err)
}

func TestHashPassword_Incorrect(t *testing.T) {
	password := "password"
	hash, _ := HashPassword(password)
	anotherPassword := "another_password"
	err := CheckPasswordHash(hash, anotherPassword)
	assert.Error(t, err)
}

func TestMakeAndValidateJWT(t *testing.T) {
	secret := "my-secret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	parsedID, err := ValidateJWT(token, secret)
	assert.NoError(t, err)
	assert.Equal(t, userID, parsedID)
}

func TestExpiredJWT(t *testing.T) {
	secret := "my-secret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, -time.Hour)
	assert.NoError(t, err)

	_, err = ValidateJWT(token, secret)
	assert.Error(t, err)
}

func TestInvalidSecret(t *testing.T) {
	secret := "correct-secret"
	wrongSecret := "wrong-secret"
	userID := uuid.New()

	token, err := MakeJWT(userID, secret, time.Hour)
	assert.NoError(t, err)

	_, err = ValidateJWT(token, wrongSecret)
	assert.Error(t, err)
}

func TestGetBearerToken_Success(t *testing.T) {
	headers := http.Header{}
	token := "some_token"
	headers.Add("Authorization", "Bearer "+token)
	token, err := GetBearerToken(headers)
	assert.NoError(t, err)
	assert.Equal(t, token, token)
}

func TestGetBearerToken_NoToken(t *testing.T) {
	headers := http.Header{}
	headers.Add("Authorization", "Another header ")
	_, err := GetBearerToken(headers)
	assert.Equal(t, err, errors.New("no token in header"))
}

func TestGetBearerToken_NoHeader(t *testing.T) {
	_, err := GetBearerToken(http.Header{})
	assert.Equal(t, err, errors.New("no authorization header"))
}

package auth

import (
	"testing"

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
	another_password := "another_password"
	err := CheckPasswordHash(hash, another_password)
	assert.Error(t, err)
}

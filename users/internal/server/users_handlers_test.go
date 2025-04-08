package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUserData_Success(t *testing.T) {
	user := userRequestBody{LoginName: "login", Email: "email@email.ru", BirthDate: "04.02.2004", Password: "password"}
	err := validateUserData(user)
	assert.NoError(t, err)
}

func TestValidateUserData_InvalidLogin(t *testing.T) {
	user := userRequestBody{LoginName: "lo", Email: "email@email.ru", BirthDate: "04.02.2004", Password: "password"}
	err := validateUserData(user)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "login name is too short")
}

func TestValidateUserData_InvalidEmail(t *testing.T) {
	user := userRequestBody{LoginName: "login", Email: "email.ru", BirthDate: "04.02.2004", Password: "password"}
	err := validateUserData(user)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid email")
}

func TestValidateUserData_InvalidPassword(t *testing.T) {
	user := userRequestBody{LoginName: "login", Email: "email@email.ru", BirthDate: "04.02.2004", Password: "pas"}
	err := validateUserData(user)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "password is too short")
}

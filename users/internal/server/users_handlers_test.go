package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUserData_InvalidLogin(t *testing.T) {
	user := RequestUser{LoginName: "lo", Email: "email@email.ru", BirthDate: "04.02.2004", Password: "password"}
	code, err := validateUserData(&ApiConfig{}, &http.Request{}, user)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "login name is too short")
	assert.Equal(t, code, http.StatusBadRequest)
}

func TestValidateUserData_InvalidEmail(t *testing.T) {
	user := RequestUser{LoginName: "login", Email: "email.ru", BirthDate: "04.02.2004", Password: "password"}
	code, err := validateUserData(&ApiConfig{}, &http.Request{}, user)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "invalid email")
	assert.Equal(t, code, http.StatusBadRequest)
}

func TestValidateUserData_InvalidPassword(t *testing.T) {
	user := RequestUser{LoginName: "login", Email: "email@email.ru", BirthDate: "04.02.2004", Password: "pas"}
	code, err := validateUserData(&ApiConfig{}, &http.Request{}, user)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "password is too short")
	assert.Equal(t, code, http.StatusBadRequest)
}

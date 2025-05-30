package server

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateUserData(t *testing.T) {
	type testCase struct {
		name               string
		user               RequestUser
		expectedError      string
		expectedStatusCode int
	}
	testCases := []testCase{
		{
			name:               "invalid_login",
			user:               RequestUser{LoginName: "lo", Email: "email@email.ru", BirthDate: "04.02.2004", Password: "password"},
			expectedError:      "login name is too short",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid_email",
			user:               RequestUser{LoginName: "login", Email: "email.ru", BirthDate: "04.02.2004", Password: "password"},
			expectedError:      "invalid email",
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:               "invalid_password",
			user:               RequestUser{LoginName: "login", Email: "email@email.ru", BirthDate: "04.02.2004", Password: "pas"},
			expectedError:      "password is too short",
			expectedStatusCode: http.StatusBadRequest,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := validateUserData(&ApiConfig{}, &http.Request{}, tc.user)
			assert.Error(t, err)
			assert.Equal(t, err.Error(), tc.expectedError)
			assert.Equal(t, code, tc.expectedStatusCode)
		})
	}
}

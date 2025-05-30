package auth

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHashPassword(t *testing.T) {
	type testCase struct {
		name            string
		password        string
		anotherPassword string
		hasError        bool
	}
	testCases := []testCase{
		{
			name:            "success",
			password:        "password",
			anotherPassword: "password",
			hasError:        false,
		},
		{
			name:            "incorrect",
			password:        "password",
			anotherPassword: "another_password",
			hasError:        true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hash, _ := HashPassword(tc.password)
			err := CheckPasswordHash(hash, tc.anotherPassword)
			assert.Equal(t, err != nil, tc.hasError)
		})
	}
}

func TestMakeAndValidateJWT(t *testing.T) {
	type testCase struct {
		name          string
		correctSecret string
		secretToCheck string
		hasError      bool
		expiresIn     time.Duration
	}
	testCases := []testCase{
		{
			name:          "valid",
			correctSecret: "my-secret",
			secretToCheck: "my-secret",
			expiresIn:     time.Hour,
			hasError:      false,
		},
		{
			name:          "expired",
			correctSecret: "my-secret",
			secretToCheck: "my-secret",
			expiresIn:     -time.Hour,
			hasError:      true,
		},
		{
			name:          "invalid_secret",
			correctSecret: "my-secret",
			secretToCheck: "wrong-secret",
			expiresIn:     time.Hour,
			hasError:      true,
		},
	}
	userID := uuid.New()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := MakeJWT(userID, tc.correctSecret, tc.expiresIn)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			parsedID, err := ValidateJWT(token, tc.secretToCheck)
			assert.Equal(t, err != nil, tc.hasError)
			if !tc.hasError {
				assert.Equal(t, userID, parsedID)
			}
		})
	}
}

func TestGetBearerToken(t *testing.T) {
	token := "some_token"
	type testCase struct {
		name          string
		headerKey     string
		headerValue   string
		expectedError error
	}
	testCases := []testCase{
		{
			name:          "success",
			headerKey:     "Authorization",
			headerValue:   "Bearer " + token,
			expectedError: nil,
		},
		{
			name:          "no_token",
			headerKey:     "Authorization",
			headerValue:   "Another header ",
			expectedError: errors.New("no token in header"),
		},
		{
			name:          "no_header",
			headerKey:     "",
			headerValue:   "",
			expectedError: errors.New("no authorization header"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := http.Header{}
			headers.Add(tc.headerKey, tc.headerValue)
			token, err := GetBearerToken(headers)
			assert.Equal(t, err, tc.expectedError)
			if tc.expectedError == nil {
				assert.Equal(t, token, token)
			}
		})
	}
}

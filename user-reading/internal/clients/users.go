package clients

import (
	"encoding/json"
	"fmt"
	"net/http"

	common "github.com/bakurvik/mylib-common"
	"github.com/google/uuid"
)

func GetUser(h http.Header, host string) (uuid.UUID, int, error) {
	client := &http.Client{}
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%v%v", host, UsersAuthWhoamiPath), nil)
	if err != nil {
		return uuid.Nil, 0, err
	}
	request.Header = h
	response, err := client.Do(request)

	if err != nil {
		return uuid.Nil, 0, err
	}
	defer common.CloseResponseBody(response)
	if response.StatusCode == http.StatusUnauthorized {
		return uuid.Nil, http.StatusUnauthorized, nil
	}
	decoder := json.NewDecoder(response.Body)
	responseData := ResponseUserID{}
	err = decoder.Decode(&responseData)
	if err != nil {
		return uuid.Nil, response.StatusCode, err
	}
	userUUID, err := uuid.Parse(responseData.ID)
	if err != nil {
		return uuid.Nil, response.StatusCode, err
	}
	return userUUID, response.StatusCode, nil
}

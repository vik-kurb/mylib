package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

func GetUser(h http.Header, host string) (uuid.UUID, int, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", fmt.Sprintf("%v%v", host, UsersAuthWhoamiPath), nil)
	if err != nil {
		return uuid.Nil, 0, err
	}
	request.Header = h
	response, err := client.Do(request)

	if err != nil {
		return uuid.Nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return uuid.Nil, http.StatusUnauthorized, nil
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return uuid.Nil, response.StatusCode, err
	}
	responseData := ResponseUserID{}
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		return uuid.Nil, response.StatusCode, err
	}
	userUUID, err := uuid.Parse(responseData.ID)
	if err != nil {
		return uuid.Nil, response.StatusCode, err
	}
	return userUUID, response.StatusCode, nil
}

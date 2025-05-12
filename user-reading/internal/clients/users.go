package clients

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

func GetUser(h http.Header, host string, port int) (uuid.UUID, int, error) {
	client := &http.Client{}
	request, err := http.NewRequest("GET", fmt.Sprint("http://%v:%v/auth/whoami", host, port), nil)
	if err != nil {
		return nil, 0, err
	}
	request.Header = h
	response, err := client.Do(request)

	if err != nil {
		return nil, 0, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return nil, http.StatusUnauthorized, nil
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, response.StatusCode, err
	}
	responseData := responseUserID{}
	err = json.Unmarshal(responseBody, &responseData)
	if err != nil {
		return nil, response.StatusCode, err
	}
	return responseData.ID, response.StatusCode, nil
}

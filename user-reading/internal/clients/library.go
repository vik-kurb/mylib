package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	common "github.com/bakurvik/mylib-common"
	"github.com/google/uuid"
)

func CheckBook(bookID uuid.UUID, host string) (int, error) {
	response, err := http.Get(fmt.Sprintf("%v%v/%v", host, LibraryApiBooksPath, bookID))
	if err != nil {
		return 0, err
	}
	defer common.CloseResponseBody(response)
	return response.StatusCode, nil
}

func GetBooksInfo(bookIDs []string, host string) (int, []ResponseBookFullInfo, error) {
	requestBook := RequestBookIDs{BookIDs: bookIDs}
	body, _ := json.Marshal(requestBook)
	response, err := http.Post(fmt.Sprintf("%v%v", host, LibraryApiBooksSearchPath), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return 0, nil, err
	}
	defer common.CloseResponseBody(response)
	decoder := json.NewDecoder(response.Body)
	responseData := []ResponseBookFullInfo{}
	err = decoder.Decode(&responseData)
	if err != nil {
		return 0, nil, err
	}
	return response.StatusCode, responseData, nil
}

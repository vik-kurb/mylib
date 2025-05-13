package clients

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func CheckBook(bookID uuid.UUID, host string) (int, error) {
	response, err := http.Get(fmt.Sprintf("http://%v/api/books/%v", host, bookID))
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	return response.StatusCode, nil
}

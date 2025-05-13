package clients

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func CheckBook(bookID uuid.UUID, host string) (int, error) {
	response, err := http.Get(fmt.Sprintf("%v%v/%v", host, LibraryApiBooksPath, bookID))
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	return response.StatusCode, nil
}

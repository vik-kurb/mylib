package clients

import (
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

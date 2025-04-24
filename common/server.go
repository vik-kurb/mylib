package common

import (
	"encoding/json"
	"log"
	"net/http"
)

func RespondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	type errorResponse struct {
		Error string `json:"error"`
	}
	responseData, err := json.Marshal(errorResponse{Error: msg})
	if err != nil {
		log.Print("Failed to build error response: ", err)
	}
	_, writeErr := w.Write(responseData)
	if writeErr != nil {
		log.Print("Failed to response: ", writeErr)
	}
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}, cookie *http.Cookie) {
	if cookie != nil {
		http.SetCookie(w, cookie)
	}
	w.WriteHeader(code)
	responseData, err := json.Marshal(payload)
	if err != nil {
		log.Print("Failed to build response: ", err)
	}
	_, writeErr := w.Write(responseData)
	if writeErr != nil {
		log.Print("Failed to response: ", writeErr)
	}
}

func CloseResponseBody(response *http.Response) {
	err := response.Body.Close()
	if err != nil {
		log.Printf("Failed to close response body: %v", err)
	}
}

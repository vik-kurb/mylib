package common

import (
	"encoding/json"
	"net/http"
)

func RespondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	type errorResponse struct {
		Error string `json:"error"`
	}
	responseData, err := json.Marshal(errorResponse{Error: msg})
	if err == nil {
		w.Write(responseData)
	}
}

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}, cookie *http.Cookie) {
	if cookie != nil {
		http.SetCookie(w, cookie)
	}
	w.WriteHeader(code)
	responseData, err := json.Marshal(payload)
	if err == nil {
		w.Write(responseData)
	}
}

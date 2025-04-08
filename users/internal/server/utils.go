package server

import (
	"encoding/json"
	"net/http"
)

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	type error_response struct {
		Error string `json:"error"`
	}
	response_data, err := json.Marshal(error_response{Error: msg})
	if err == nil {
		w.Write(response_data)
	}
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}, cookie *http.Cookie) {
	if cookie != nil {
		http.SetCookie(w, cookie)
	}
	w.WriteHeader(code)
	response_data, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Write(response_data)
}

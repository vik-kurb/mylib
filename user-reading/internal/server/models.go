package server

type RequestUserReading struct {
	BookId string `json:"book_id"`
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

package server

type UserReading struct {
	BookID string `json:"book_id"`
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type UserReadingFullInfo struct {
	BookID
	Title
	Authors
}

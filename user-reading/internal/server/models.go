package server

type UserReading struct {
	BookID string `json:"book_id"`
	Status string `json:"status"`
	Rating int
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ResponseUserReading struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	Status  string   `json:"status"`
}

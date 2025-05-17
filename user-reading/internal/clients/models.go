package clients

const (
	UsersAuthWhoamiPath       = "/auth/whoami"
	LibraryApiBooksPath       = "/api/books"
	LibraryApiBooksSearchPath = "/api/books/search"
)

type ResponseUserID struct {
	ID string `json:"user_id"`
}

type ResponseBookFullInfo struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type RequestBookIDs struct {
	BookIDs []string `json:"book_ids"`
}

package clients

const (
	UsersAuthWhoamiPath = "/auth/whoami"
	LibraryApiBooksPath = "/api/books"
)

type ResponseUserID struct {
	ID string `json:"user_id"`
}

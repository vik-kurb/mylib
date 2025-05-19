package server

type RequestAuthor struct {
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date,omitempty"`
	DeathDate string `json:"death_date,omitempty"`
}

type RequestAuthorWithID struct {
	ID        string `json:"id"`
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date,omitempty"`
	DeathDate string `json:"death_date,omitempty"`
}

type ResponseAuthorShortInfo struct {
	FullName string `json:"full_name"`
	ID       string `json:"id"`
}

type ResponseAuthorFullInfo struct {
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date,omitempty"`
	DeathDate string `json:"death_date,omitempty"`
}

type ResponseBook struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type RequestBook struct {
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type RequestBookWithID struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type ResponseBookFullInfo struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type RequestBookIDs struct {
	BookIDs []string `json:"book_ids"`
}

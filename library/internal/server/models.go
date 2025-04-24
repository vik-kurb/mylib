package server

type RequestAuthor struct {
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date,omitempty"`
	DeathDate string `json:"death_date,omitempty"`
}

type RequestAuthorWithID struct {
	Id        string `json:"id"`
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date,omitempty"`
	DeathDate string `json:"death_date,omitempty"`
}

type ResponseAuthorShortInfo struct {
	FullName string `json:"full_name"`
	Id       string `json:"id"`
}

type ResponseAuthorFullInfo struct {
	FullName  string `json:"full_name"`
	BirthDate string `json:"birth_date,omitempty"`
	DeathDate string `json:"death_date,omitempty"`
}

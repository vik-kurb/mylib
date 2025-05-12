package server

type RequestLogin struct {
	Password string `json:"password"`
	Email    string `json:"email"`
}

type RequestUser struct {
	LoginName string `json:"login_name"`
	Email     string `json:"email"`
	BirthDate string `json:"birth_date,omitempty"`
	Password  string `json:"password"`
}

type ResponseToken struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type ResponseUser struct {
	LoginName string `json:"login"`
	Email     string `json:"email"`
	BirthDate string `json:"birth_date"`
}

type ResponseUserID struct {
	ID string `json:"user_id"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

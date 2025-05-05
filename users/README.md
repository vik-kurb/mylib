## Users API:

### POST /api/users
Creates new user and stores it in DB

### PUT /api/users
Updates existing user's info in DB. Uses refresh token from an HTTP-only cookie

### GET /api/users/{id}
Gets user from DB

### DELETE /api/users
Deletes user from DB. Uses refresh token from an HTTP-only cookie

## Auth API:

### POST /api/login
Checks password and returns access and refresh tokens

### POST /api/refresh
Checks refresh token from an HTTP-only cookie and returns new access and refresh tokens

### POST /api/revoke
Revokes refresh token from an HTTP-only cookie

## Health API:

### GET /ping
Checks server health. Returns 200 OK if server is up
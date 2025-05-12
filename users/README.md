## Users API:

### POST /api/users
Creates new user and stores it in DB

### PUT /api/users
Updates existing user's info in DB. Uses access token from an HTTP-only cookie

### GET /api/users/{id}
Gets user from DB

### DELETE /api/users
Deletes user from DB. Uses access token from an HTTP-only cookie

## Auth API:

### POST /auth/login
Checks password and returns access and refresh tokens

### POST /auth/refresh
Checks refresh token from an HTTP-only cookie and returns new access and refresh tokens

### POST /auth/revoke
Revokes refresh token from an HTTP-only cookie

### POST /auth/whoami
Gets user ID. Uses access token from an HTTP-only cookie

## Health API:

### GET /ping
Checks server health. Returns 200 OK if server is up
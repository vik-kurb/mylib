## users
Microservice that stores users data.

Environment variables should be set in .env:
| Variable          | Description                                             | Example                                                            |
| ----------------- | ------------------------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`         | Name of the main application database                   | `library`                                                          |
| `DB_HOST`         | Hostname of the PostgreSQL server                       | `db` (Docker service name)                                         |
| `DB_PORT`         | Port on which PostgreSQL is listening                   | `5432`                                                             |
| `DB_USER`         | Database user                                           | `postgres`                                                         |
| `DB_PASSWORD`     | Database user password                                  | `postgres`                                                         |
| `TEST_DB_URL`     | Connection URL for test database (local)                | `postgres://postgres:@localhost:5432/test_library?sslmode=disable` |
| `AUTH_SECRET_KEY` | Secret key used for signing and verifying JWT tokens    | `Q4uTGasVKJUqlpvhlpQ/Lkg3i+3z5LLdkUPH2tjO1dEVWUqnb9VGjPBhV2rAXh63` |
| `CORS_ALLOWED_ORIGIN`      | Allowed origin for cross-origin HTTP requests (Access-Control-Allow-Origin response header in CORS middleware) | `http://localhost:5173/` |

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
## user-reading
Microservice that stores reading status of user books.

Environment variables should be set in .env:
| Variable      | Description                              | Example                                                            |
| ------------- | ---------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`     | Name of the main application database    | `user_reading`                                                          |
| `DB_HOST`     | Hostname of the PostgreSQL server        | `db` (Docker service name)                                         |
| `DB_PORT`     | Port on which PostgreSQL is listening    | `5432`                                                             |
| `DB_USER`     | Database user                            | `postgres`                                                         |
| `DB_PASSWORD` | Database user password                   | `postgres`                                                         |
| `TEST_DB_URL` | Connection URL for test database (local) | `postgres://postgres:@localhost:5432/test_user_reading?sslmode=disable` |
| `USERS_SERVICE_HOST` | Host of users service | `http://users:8080` |
| `LIBRARY_SERVICE_HOST` | Host of library service | `http://library:8080` |
| `CORS_ALLOWED_ORIGIN`      | Allowed origin for cross-origin HTTP requests (Access-Control-Allow-Origin response header in CORS middleware) | `http://localhost:5173/` |
| `LIBRARY_BOOKS_CACHE_ENABLE` | Enable cache of books from library service | `false` |
| `LIBRARY_BOOKS_CACHE_CLEANUP_PERIOD_MIN` | Cleanup period of books cache (minutes) | `60` |
| `LIBRARY_BOOKS_CACHE_CLEANUP_OLD_THRESHOLD_MIN` | Threshold for deleting old data in books cache (minutes) | `60` |

## User reading API:

### GET /ping
Checks server health. Returns 200 OK if server is up

### POST /api/user-reading
Saves book to user reading in DB. Uses access token from an HTTP-only cookie

### PUT /api/user-reading
Updates user reading in DB. Uses access token from an HTTP-only cookie

### DELETE /api/user-reading/{bookID}
Deletes user reading from DB. Uses access token from an HTTP-only cookie

### GET /api/user-reading
Gets user reading from DB. Uses access token from an HTTP-only cookie

### GET /api/user-reading/{bookID}
Gets user reading full info from DB. Uses access token from an HTTP-only cookie
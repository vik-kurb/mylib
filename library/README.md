## library
Microservice that stores books and authors data.

Environment variables should be set in .env:
| Variable                   | Description                               | Example                                                            |
| -------------------------- | ----------------------------------------- | -------------------------------------------------------------------|
| `DB_NAME`                  | Name of the main application database     | `library`                                                          |
| `DB_HOST`                  | Hostname of the PostgreSQL server         | `db` (Docker service name)                                         |
| `DB_PORT`                  | Port on which PostgreSQL is listening     | `5432`                                                             |
| `DB_USER`                  | Database user                             | `postgres`                                                         |
| `DB_PASSWORD`              | Database user password                    | `postgres`                                                         |
| `TEST_DB_URL`              | Connection URL for test database (local)  | `postgres://postgres:@localhost:5432/test_library?sslmode=disable` |
| `MAX_SEARCH_BOOKS_LIMIT`   | Maximum number of books found in search   | `10`                                                               |
| `MAX_SEARCH_AUTHORS_LIMIT` | Maximum number of authors found in search | `10`                                                               |
| `CORS_ALLOWED_ORIGIN`      | Allowed origin for cross-origin HTTP requests (Access-Control-Allow-Origin response header in CORS middleware) | `http://localhost:5173/` |

## Authors API:

### POST /api/authors
Creates new author and stores it in DB

### GET /api/authors
Gets all authors from DB

### GET /api/authors/{id}
Gets an author with requested ID from DB

### DELETE /admin/authors/{id}
Deletes an author with requested ID from DB

### PUT /api/authors
Updates existing author's info in DB

### GET /api/authors/{id}/books
Returns a list of books written by the specified author

### GET /api/authors/search
Searches authors by name. Uses postgres full text search

## Books API:

### POST /api/books
Creates new book and stores it in DB

### PUT /api/books
Updates existing book's info in DB

### GET /api/books
Gets books with requested ID from DB

### POST /admin/books/{id}
Deletes a book from DB with requested ID

### GET /api/books/search
Searches books by title. Uses postgres full text search

## Health API:

### GET /ping
Checks server health. Returns 200 OK if server is up
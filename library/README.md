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
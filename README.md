# mylib


## library
Microservice that stores books and authors data. API:

### POST /api/authors
Creates new author

### GET /api/authors
Gets all authors with short info

### GET /api/authors/{id}
Gets an author full info

### DELETE /admin/authors/{id}
Deletes an author

### PUT /api/authors
Update an author info


## users
Microservice that stores users data. API:

### POST /api/users
Creates new user

### POST /api/login
Logins user

### POST /api/refresh
Refreshes access token

### POST /api/revoke
Revokes refresh token

### PUT /api/users
Update user info

### GET /api/users/{id}
Gets user info

### DELETE /api/users
Deletes user

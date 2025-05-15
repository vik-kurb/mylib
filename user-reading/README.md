## User reading API:

### GET /ping
Checks server health. Returns 200 OK if server is up

### POST /api/user-reading
Saves book to user reading in DB. Uses access token from an HTTP-only cookie

### PUT /api/user-reading
Updates user reading in DB. Uses access token from an HTTP-only cookie
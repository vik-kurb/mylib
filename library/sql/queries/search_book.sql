SELECT id, title, ts_rank(tsv, plainto_tsquery('english', $1)) AS rank
FROM books WHERE tsv @@ plainto_tsquery('english', $1)
ORDER BY rank DESC
LIMIT $2;

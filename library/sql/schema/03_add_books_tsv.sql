-- +goose Up
ALTER TABLE books ADD COLUMN tsv tsvector;

-- +goose StatementBegin
CREATE FUNCTION books_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.tsv := to_tsvector('english', NEW.title);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trigger_books_tsv
BEFORE INSERT OR UPDATE ON books
FOR EACH ROW EXECUTE FUNCTION books_tsv_trigger();

CREATE INDEX idx_books_tsv ON books USING GIN(tsv);

-- +goose Down
DROP TRIGGER IF EXISTS trigger_books_tsv ON books;

DROP FUNCTION IF EXISTS books_tsv_trigger();

DROP INDEX IF EXISTS idx_books_tsv;

ALTER TABLE books DROP COLUMN IF EXISTS tsv;

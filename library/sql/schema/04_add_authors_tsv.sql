-- +goose Up
ALTER TABLE authors ADD COLUMN tsv tsvector;

-- +goose StatementBegin
CREATE FUNCTION authors_tsv_trigger() RETURNS trigger AS $$
BEGIN
  NEW.tsv := to_tsvector('english', NEW.full_name);
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trigger_authors_tsv
BEFORE INSERT OR UPDATE ON authors
FOR EACH ROW EXECUTE FUNCTION authors_tsv_trigger();

CREATE INDEX idx_authors_tsv ON authors USING GIN(tsv);

-- +goose Down
DROP TRIGGER IF EXISTS trigger_authors_tsv ON authors;

DROP FUNCTION IF EXISTS authors_tsv_trigger();

DROP INDEX IF EXISTS idx_authors_tsv;

ALTER TABLE authors DROP COLUMN IF EXISTS tsv;

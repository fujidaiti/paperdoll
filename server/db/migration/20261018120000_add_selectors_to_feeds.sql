-- +goose Up
-- The extraction rules of an unstructured feed, as a JSON object of
-- feed.ExtractionRules.
ALTER TABLE feeds ADD COLUMN extraction_rules json;

-- +goose Down
ALTER TABLE feeds DROP COLUMN extraction_rules;

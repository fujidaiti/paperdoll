-- +goose Up
-- The post selectors of an unstructured feed, as a JSON array of
-- feed.PostSelector.
ALTER TABLE feeds ADD COLUMN selectors json;

-- +goose Down
ALTER TABLE feeds DROP COLUMN selectors;

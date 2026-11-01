-- +goose Up
-- The post selectors of an unstructured feed (see feed.PostSelector), as a JSON
-- array kept as the client sent it.
ALTER TABLE feeds ADD COLUMN selectors text;

-- +goose Down
ALTER TABLE feeds DROP COLUMN selectors;

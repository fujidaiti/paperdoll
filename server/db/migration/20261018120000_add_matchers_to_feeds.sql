-- +goose Up
-- The post matchers of an unstructured feed (see feed.PostMatcher), as a JSON
-- array kept as the client sent it.
ALTER TABLE feeds ADD COLUMN matchers text;

-- +goose Down
ALTER TABLE feeds DROP COLUMN matchers;

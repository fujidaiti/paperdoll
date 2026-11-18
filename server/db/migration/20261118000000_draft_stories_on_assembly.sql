-- +goose Up
-- Stories are drafted when a newspaper is assembled, so every story belongs to
-- a newspaper. Unclaimed stories are dropped. The assembly job drafts them again
-- if their entries are within the editorial interval of its next newspaper.
DELETE FROM stories WHERE newspaper_id IS NULL;
ALTER TABLE stories ALTER COLUMN newspaper_id SET NOT NULL;

-- +goose Down
ALTER TABLE stories ALTER COLUMN newspaper_id DROP NOT NULL;

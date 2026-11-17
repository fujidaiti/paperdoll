-- +goose Up
-- Stories are drafted when a newspaper is assembled, so every story belongs to
-- a newspaper. Unclaimed stories are dropped. The assembly job drafts them again
-- if their entries are still within its window.
DELETE FROM stories WHERE newspaper_id IS NULL;
ALTER TABLE stories ALTER COLUMN newspaper_id SET NOT NULL;

-- A user gets at most one story from each feed entry.
ALTER TABLE stories ADD CONSTRAINT uq_story_per_user_entry UNIQUE (user_id, feed_entry_id);

-- +goose Down
ALTER TABLE stories DROP CONSTRAINT uq_story_per_user_entry;
ALTER TABLE stories ALTER COLUMN newspaper_id DROP NOT NULL;

-- +goose Up
-- A temporary editor in which a user edits the post selectors of an
-- unstructured feed (see feed.SelectorEditor).
CREATE TABLE selector_editors (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id int NOT NULL REFERENCES users (id),
    feed_id bigint NOT NULL REFERENCES feeds (id),
    -- The simplified page with a data-id on each element.
    simplified_page text NOT NULL,
    -- The labeled posts, as a JSON array of
    -- {"box": id, "title": id, "link": id, ...}. They start as the posts that
    -- the saved selectors find.
    labeled_posts json NOT NULL,
    -- The host and path of each excluded link.
    excluded_links text[] NOT NULL,
    -- The node ID of each node that is excluded with its subtree.
    excluded_nodes int[] NOT NULL,
    -- The extraction rules generated from the marks, as a JSON object of
    -- feed.ExtractionRules.
    extraction_rules json NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX idx_selector_editors_expires_at ON selector_editors (expires_at);

-- +goose Down
DROP TABLE selector_editors;

-- +goose Up
CREATE TABLE feeds (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    url text NOT NULL UNIQUE,
    site_url text,
    icon_url text,
    title text NOT NULL,
    description text
);

CREATE TABLE feed_entries (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dedup_key text NOT NULL UNIQUE,
    feed_id bigint NOT NULL REFERENCES feeds (id),
    url text NOT NULL,
    title text NOT NULL,
    description text,
    content text,
    snapshot_at timestamptz NOT NULL,
    published_at timestamptz
);

CREATE TABLE newspapers (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    published_at timestamptz NOT NULL UNIQUE,
    draft boolean NOT NULL DEFAULT true
);

CREATE TABLE stories (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    feed_entry_id bigint NOT NULL REFERENCES feed_entries (id),
    newspaper_id bigint REFERENCES newspapers (id),
    title text NOT NULL,
    description text,
    published_at timestamptz,
    source text
);

CREATE TABLE newspaper_schedules (
    id int GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    label text NOT NULL,
    -- 1439min = 60min * 24h - 1min = 23h 59min
    minute_of_date integer NOT NULL CHECK (minute_of_date BETWEEN 0 AND 1439)
);

CREATE TABLE web_clips (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  url text NOT NULL,
  title text,
  description text,
  content text,
  fetch_status text NOT NULL DEFAULT 'pending'
    CHECK (fetch_status IN ('pending', 'done', 'failed')),
  updated_at timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT chk_content_when_done CHECK (
    (fetch_status = 'done' AND content IS NOT NULL) OR
    (fetch_status != 'done' AND content IS NULL)
  )
);

CREATE TABLE reading_list_items (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  kind          text NOT NULL CHECK (kind IN ('feed_entry', 'web_clip')),
  web_clip_id bigint REFERENCES web_clips (id), -- TODO(#106): Make this unique
  feed_entry_id bigint REFERENCES feed_entries (id), -- TODO(#106): Make this unique
  title         text NOT NULL, -- TODO(#108): Make this nullable
  description   text,
  archived      boolean NOT NULL DEFAULT false,
  saved_at      timestamptz NOT NULL DEFAULT now(),

  CONSTRAINT chk_kind_and_id CHECK (
    (kind = 'feed_entry' AND feed_entry_id IS NOT NULL AND web_clip_id IS NULL) OR
    (kind = 'web_clip' AND feed_entry_id IS NULL AND web_clip_id is NOT NULL)
  )
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS trigger AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER trg_web_clips_updated_at
BEFORE UPDATE ON web_clips
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

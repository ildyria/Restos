-- +goose Up
-- Site-wide values edited in the admin panel, such as the About page text.
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- +goose Down
DROP TABLE settings;

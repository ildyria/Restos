-- +goose Up
-- Photos belong to a restaurant, usually through a visit. The dish (what the photo shows) is
-- optional, and deleting a dish keeps its photos. A photo without a visit is the restaurant's
-- picture taken from its website (source = 'website'), at most one per restaurant.
CREATE TABLE photos_new (
    id            INTEGER PRIMARY KEY,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    visit_id      INTEGER REFERENCES visits(id) ON DELETE CASCADE,
    dish_id       INTEGER REFERENCES dishes(id) ON DELETE SET NULL,
    source        TEXT NOT NULL DEFAULT 'upload' CHECK (source IN ('upload', 'website')),
    file_key      TEXT NOT NULL UNIQUE,
    width         INTEGER NOT NULL,
    height        INTEGER NOT NULL,
    position      INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    CHECK ((source = 'website') = (visit_id IS NULL))
);
INSERT INTO photos_new(id, restaurant_id, visit_id, dish_id, file_key, width, height, position, created_at)
SELECT p.id, v.restaurant_id, d.visit_id, p.dish_id, p.file_key, p.width, p.height, p.position, p.created_at
FROM photos p JOIN dishes d ON d.id = p.dish_id JOIN visits v ON v.id = d.visit_id;
DROP TABLE photos;
ALTER TABLE photos_new RENAME TO photos;
CREATE INDEX photos_restaurant ON photos(restaurant_id);
CREATE INDEX photos_visit ON photos(visit_id, position);
CREATE INDEX photos_dish ON photos(dish_id);
CREATE UNIQUE INDEX photos_one_website ON photos(restaurant_id) WHERE source = 'website';

-- Images fetched from a website before the restaurant is saved; claimed on save, purged when stale.
CREATE TABLE pending_media (
    file_key   TEXT PRIMARY KEY,
    width      INTEGER NOT NULL,
    height     INTEGER NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

-- +goose Down
-- Photos without a dish (including website pictures) cannot be represented in the old schema
-- and are dropped; their files stay on disk.
DROP TABLE pending_media;
CREATE TABLE photos_old (
    id         INTEGER PRIMARY KEY,
    dish_id    INTEGER NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    file_key   TEXT NOT NULL UNIQUE,
    width      INTEGER NOT NULL,
    height     INTEGER NOT NULL,
    position   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
INSERT INTO photos_old(id, dish_id, file_key, width, height, position, created_at)
SELECT id, dish_id, file_key, width, height, position, created_at FROM photos WHERE dish_id IS NOT NULL;
DROP TABLE photos;
ALTER TABLE photos_old RENAME TO photos;
CREATE INDEX photos_dish ON photos(dish_id, position);
UPDATE restaurants SET cover_photo_id = NULL WHERE cover_photo_id NOT IN (SELECT id FROM photos);

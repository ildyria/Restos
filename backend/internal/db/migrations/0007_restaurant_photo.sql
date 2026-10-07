-- +goose Up
-- A restaurant can have its own photo (source = 'restaurant'): uploaded, attached to no visit and
-- showing no dish, at most one per restaurant. SQLite cannot change a CHECK, so the table is rebuilt.
CREATE TABLE photos_new (
    id            INTEGER PRIMARY KEY,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    visit_id      INTEGER REFERENCES visits(id) ON DELETE CASCADE,
    dish_id       INTEGER REFERENCES dishes(id) ON DELETE SET NULL,
    source        TEXT NOT NULL DEFAULT 'upload' CHECK (source IN ('upload', 'website', 'restaurant')),
    file_key      TEXT NOT NULL UNIQUE,
    width         INTEGER NOT NULL,
    height        INTEGER NOT NULL,
    position      INTEGER NOT NULL DEFAULT 0,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    CHECK ((source = 'upload') = (visit_id IS NOT NULL)),
    CHECK (source = 'upload' OR dish_id IS NULL)
);
INSERT INTO photos_new(id, restaurant_id, visit_id, dish_id, source, file_key, width, height, position, created_at)
SELECT id, restaurant_id, visit_id, dish_id, source, file_key, width, height, position, created_at FROM photos;
DROP TABLE photos;
ALTER TABLE photos_new RENAME TO photos;
CREATE INDEX photos_restaurant ON photos(restaurant_id);
CREATE INDEX photos_visit ON photos(visit_id, position);
CREATE INDEX photos_dish ON photos(dish_id);
CREATE UNIQUE INDEX photos_one_website ON photos(restaurant_id) WHERE source = 'website';
CREATE UNIQUE INDEX photos_one_restaurant ON photos(restaurant_id) WHERE source = 'restaurant';

-- +goose Down
-- Restaurant photos have no place in the old schema and are dropped; their files stay on disk.
CREATE TABLE photos_old (
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
INSERT INTO photos_old(id, restaurant_id, visit_id, dish_id, source, file_key, width, height, position, created_at)
SELECT id, restaurant_id, visit_id, dish_id, source, file_key, width, height, position, created_at FROM photos
WHERE source <> 'restaurant';
DROP TABLE photos;
ALTER TABLE photos_old RENAME TO photos;
CREATE INDEX photos_restaurant ON photos(restaurant_id);
CREATE INDEX photos_visit ON photos(visit_id, position);
CREATE INDEX photos_dish ON photos(dish_id);
CREATE UNIQUE INDEX photos_one_website ON photos(restaurant_id) WHERE source = 'website';
UPDATE restaurants SET cover_photo_id = NULL WHERE cover_photo_id NOT IN (SELECT id FROM photos);

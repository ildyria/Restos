-- +goose Up
CREATE TABLE users (
    id            INTEGER PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL
);

CREATE TABLE restaurants (
    id              INTEGER PRIMARY KEY,
    slug            TEXT NOT NULL UNIQUE,
    name            TEXT NOT NULL,
    cuisine         TEXT NOT NULL DEFAULT '',
    address         TEXT NOT NULL DEFAULT '',
    city            TEXT NOT NULL DEFAULT '',
    website         TEXT NOT NULL DEFAULT '',
    price_level     INTEGER CHECK (price_level BETWEEN 1 AND 4),
    rating_food     REAL CHECK (rating_food BETWEEN 1 AND 5),
    rating_service  REAL CHECK (rating_service BETWEEN 1 AND 5),
    rating_ambiance REAL CHECK (rating_ambiance BETWEEN 1 AND 5),
    rating_value    REAL CHECK (rating_value BETWEEN 1 AND 5),
    review          TEXT NOT NULL DEFAULT '',
    rank_position   INTEGER,
    cover_photo_id  INTEGER,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
    updated_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE visits (
    id            INTEGER PRIMARY KEY,
    restaurant_id INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    visited_on    TEXT NOT NULL,
    meal          TEXT NOT NULL CHECK (meal IN ('lunch','dinner')),
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX visits_restaurant ON visits(restaurant_id, visited_on DESC);

CREATE TABLE dishes (
    id       INTEGER PRIMARY KEY,
    visit_id INTEGER NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
    name     TEXT NOT NULL,
    course   TEXT CHECK (course IN ('starter','main','dessert','drink','other')),
    notes    TEXT NOT NULL DEFAULT '',
    position INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX dishes_visit ON dishes(visit_id, position);

CREATE TABLE photos (
    id         INTEGER PRIMARY KEY,
    dish_id    INTEGER NOT NULL REFERENCES dishes(id) ON DELETE CASCADE,
    file_key   TEXT NOT NULL UNIQUE,
    width      INTEGER NOT NULL,
    height     INTEGER NOT NULL,
    position   INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);
CREATE INDEX photos_dish ON photos(dish_id, position);

-- +goose Down
DROP TABLE photos;
DROP TABLE dishes;
DROP TABLE visits;
DROP TABLE restaurants;
DROP TABLE sessions;
DROP TABLE users;

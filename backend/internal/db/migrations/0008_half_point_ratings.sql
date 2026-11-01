-- +goose Up
-- Ratings may start at half a point (0.5 to 5 in half steps). SQLite cannot change a CHECK, so the
-- scores table is rebuilt.
CREATE TABLE scores_new (
    restaurant_id   INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating_food     REAL CHECK (rating_food BETWEEN 0.5 AND 5),
    rating_service  REAL CHECK (rating_service BETWEEN 0.5 AND 5),
    rating_ambiance REAL CHECK (rating_ambiance BETWEEN 0.5 AND 5),
    rating_value    REAL CHECK (rating_value BETWEEN 0.5 AND 5),
    would_return    INTEGER CHECK (would_return IN (0, 1)),
    rating_quiet    REAL CHECK (rating_quiet BETWEEN 0.5 AND 5),
    PRIMARY KEY (restaurant_id, user_id)
);
INSERT INTO scores_new(restaurant_id, user_id, rating_food, rating_service, rating_ambiance, rating_value, would_return, rating_quiet)
SELECT restaurant_id, user_id, rating_food, rating_service, rating_ambiance, rating_value, would_return, rating_quiet FROM scores;
DROP TABLE scores;
ALTER TABLE scores_new RENAME TO scores;
CREATE INDEX scores_user ON scores(user_id);

-- +goose Down
-- Half-point ratings are raised to 1, the old minimum (MAX keeps NULL as NULL).
CREATE TABLE scores_old (
    restaurant_id   INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating_food     REAL CHECK (rating_food BETWEEN 1 AND 5),
    rating_service  REAL CHECK (rating_service BETWEEN 1 AND 5),
    rating_ambiance REAL CHECK (rating_ambiance BETWEEN 1 AND 5),
    rating_value    REAL CHECK (rating_value BETWEEN 1 AND 5),
    would_return    INTEGER CHECK (would_return IN (0, 1)),
    rating_quiet    REAL CHECK (rating_quiet BETWEEN 1 AND 5),
    PRIMARY KEY (restaurant_id, user_id)
);
INSERT INTO scores_old(restaurant_id, user_id, rating_food, rating_service, rating_ambiance, rating_value, would_return, rating_quiet)
SELECT restaurant_id, user_id, MAX(rating_food, 1), MAX(rating_service, 1), MAX(rating_ambiance, 1), MAX(rating_value, 1),
       would_return, MAX(rating_quiet, 1)
FROM scores;
DROP TABLE scores;
ALTER TABLE scores_old RENAME TO scores;
CREATE INDEX scores_user ON scores(user_id);

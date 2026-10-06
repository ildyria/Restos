-- +goose Up
-- Each person (user) scores a restaurant separately; the public scores are their averages.
CREATE TABLE scores (
    restaurant_id   INTEGER NOT NULL REFERENCES restaurants(id) ON DELETE CASCADE,
    user_id         INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rating_food     REAL CHECK (rating_food BETWEEN 1 AND 5),
    rating_service  REAL CHECK (rating_service BETWEEN 1 AND 5),
    rating_ambiance REAL CHECK (rating_ambiance BETWEEN 1 AND 5),
    rating_value    REAL CHECK (rating_value BETWEEN 1 AND 5),
    would_return    INTEGER CHECK (would_return IN (0, 1)),
    PRIMARY KEY (restaurant_id, user_id)
);
CREATE INDEX scores_user ON scores(user_id);

-- Existing single ratings are attributed to the first user, if there is one.
INSERT INTO scores(restaurant_id, user_id, rating_food, rating_service, rating_ambiance, rating_value)
SELECT r.id, (SELECT MIN(id) FROM users), r.rating_food, r.rating_service, r.rating_ambiance, r.rating_value
FROM restaurants r
WHERE EXISTS (SELECT 1 FROM users)
  AND COALESCE(r.rating_food, r.rating_service, r.rating_ambiance, r.rating_value) IS NOT NULL;

ALTER TABLE restaurants DROP COLUMN rating_food;
ALTER TABLE restaurants DROP COLUMN rating_service;
ALTER TABLE restaurants DROP COLUMN rating_ambiance;
ALTER TABLE restaurants DROP COLUMN rating_value;

ALTER TABLE restaurants ADD COLUMN latitude REAL CHECK (latitude BETWEEN -90 AND 90);
ALTER TABLE restaurants ADD COLUMN longitude REAL CHECK (longitude BETWEEN -180 AND 180);

-- +goose Down
ALTER TABLE restaurants DROP COLUMN latitude;
ALTER TABLE restaurants DROP COLUMN longitude;
ALTER TABLE restaurants ADD COLUMN rating_food REAL CHECK (rating_food BETWEEN 1 AND 5);
ALTER TABLE restaurants ADD COLUMN rating_service REAL CHECK (rating_service BETWEEN 1 AND 5);
ALTER TABLE restaurants ADD COLUMN rating_ambiance REAL CHECK (rating_ambiance BETWEEN 1 AND 5);
ALTER TABLE restaurants ADD COLUMN rating_value REAL CHECK (rating_value BETWEEN 1 AND 5);
UPDATE restaurants SET
    rating_food     = (SELECT AVG(rating_food) FROM scores WHERE restaurant_id = restaurants.id),
    rating_service  = (SELECT AVG(rating_service) FROM scores WHERE restaurant_id = restaurants.id),
    rating_ambiance = (SELECT AVG(rating_ambiance) FROM scores WHERE restaurant_id = restaurants.id),
    rating_value    = (SELECT AVG(rating_value) FROM scores WHERE restaurant_id = restaurants.id);
DROP TABLE scores;

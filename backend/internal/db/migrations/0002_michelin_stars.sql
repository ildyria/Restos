-- +goose Up
ALTER TABLE restaurants ADD COLUMN michelin_stars INTEGER CHECK (michelin_stars BETWEEN 1 AND 3);

-- +goose Down
ALTER TABLE restaurants DROP COLUMN michelin_stars;

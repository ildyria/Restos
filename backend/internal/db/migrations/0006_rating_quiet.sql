-- +goose Up
-- How quiet the room was: 5 is calm enough to talk easily, 1 is shouting over the noise.
ALTER TABLE scores ADD COLUMN rating_quiet REAL CHECK (rating_quiet BETWEEN 1 AND 5);

-- +goose Down
ALTER TABLE scores DROP COLUMN rating_quiet;

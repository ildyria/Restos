package db

import (
	"database/sql"
	"errors"
	"time"
)

// ErrUnknownMedia is returned when a pending image key is unknown (never fetched, or already purged).
var ErrUnknownMedia = errors.New("unknown or expired image")

// AddPendingMedia records an image fetched before its restaurant is saved.
func (s *Store) AddPendingMedia(key string, w, h int) error {
	_, err := s.DB.Exec(`INSERT INTO pending_media(file_key, width, height) VALUES(?,?,?)`, key, w, h)
	return err
}

// PurgePendingMedia forgets pending images older than the cutoff and returns their keys so the
// caller can delete the files.
func (s *Store) PurgePendingMedia(olderThan time.Time) ([]string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	cutoff := olderThan.UTC().Format("2006-01-02T15:04:05Z")
	keys, err := queryKeys(tx, `SELECT file_key FROM pending_media WHERE created_at < ?`, cutoff)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM pending_media WHERE created_at < ?`, cutoff); err != nil {
		return nil, err
	}
	return keys, tx.Commit()
}

// SetSitePhoto makes a pending image the restaurant's website picture ("" removes it), replacing
// any previous one. A cover pointing at the old picture moves to the new one. It returns the file
// keys that are no longer used.
func (s *Store) SetSitePhoto(restaurantID int64, key string) ([]string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var oldID int64
	var oldKey string
	err = tx.QueryRow(`SELECT id, file_key FROM photos WHERE restaurant_id = ? AND source = 'website'`, restaurantID).Scan(&oldID, &oldKey)
	hasOld := err == nil
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if hasOld && oldKey == key {
		return nil, nil // saved again with the picture it already has
	}

	var removed []string
	if hasOld {
		if _, err := tx.Exec(`DELETE FROM photos WHERE id = ?`, oldID); err != nil {
			return nil, err
		}
		removed = append(removed, oldKey)
	}
	if key != "" {
		var w, h int
		if err := tx.QueryRow(`SELECT width, height FROM pending_media WHERE file_key = ?`, key).Scan(&w, &h); err != nil {
			if err == sql.ErrNoRows {
				return nil, ErrUnknownMedia
			}
			return nil, err
		}
		if _, err := tx.Exec(`DELETE FROM pending_media WHERE file_key = ?`, key); err != nil {
			return nil, err
		}
		res, err := tx.Exec(`INSERT INTO photos(restaurant_id, source, file_key, width, height) VALUES(?, 'website', ?, ?, ?)`,
			restaurantID, key, w, h)
		if err != nil {
			return nil, err
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return nil, err
		}
		if hasOld {
			if _, err := tx.Exec(`UPDATE restaurants SET cover_photo_id = ? WHERE id = ? AND cover_photo_id = ?`, newID, restaurantID, oldID); err != nil {
				return nil, err
			}
		}
	}
	if err := clearDanglingCovers(tx); err != nil {
		return nil, err
	}
	return removed, tx.Commit()
}

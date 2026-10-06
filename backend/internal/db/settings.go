package db

import "database/sql"

// Setting returns a site-wide value; ok is false when it was never set.
func (s *Store) Setting(key string) (value string, ok bool, err error) {
	err = s.DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return value, err == nil, err
}

// SetSetting stores a site-wide value, replacing any previous one.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.DB.Exec(`INSERT INTO settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// DeleteSetting forgets a site-wide value, so its default applies again.
func (s *Store) DeleteSetting(key string) error {
	_, err := s.DB.Exec(`DELETE FROM settings WHERE key = ?`, key)
	return err
}

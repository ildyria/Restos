package db

import (
	"errors"
	"strings"
	"time"
)

type User struct {
	ID           int64  `json:"id"`
	Email        string `json:"email"`
	Name         string `json:"name"`
	PasswordHash string `json:"-"`
	CreatedAt    string `json:"created_at"`
}

var ErrEmailTaken = errors.New("email already in use")
var ErrLastUser = errors.New("cannot delete the last remaining user")

func (s *Store) CreateUser(email, name, hash string) (User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	res, err := s.DB.Exec(`INSERT INTO users(email, name, password_hash) VALUES(?,?,?)`, email, name, hash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	id, _ := res.LastInsertId()
	return s.UserByID(id)
}

func (s *Store) UserByID(id int64) (User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, email, name, password_hash, created_at FROM users WHERE id=?`, id).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	return u, notFound(err)
}

func (s *Store) UserByEmail(email string) (User, error) {
	var u User
	err := s.DB.QueryRow(`SELECT id, email, name, password_hash, created_at FROM users WHERE email=?`,
		strings.ToLower(strings.TrimSpace(email))).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt)
	return u, notFound(err)
}

func (s *Store) ListUsers() ([]User, error) {
	rows, err := s.DB.Query(`SELECT id, email, name, password_hash, created_at FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (s *Store) DeleteUser(id int64) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastUser
	}
	res, err := tx.Exec(`DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return err
	}
	if err := affected(res); err != nil {
		return err
	}
	return tx.Commit()
}

// Sessions

func (s *Store) CreateSession(tokenHash string, userID int64, expires time.Time) error {
	_, err := s.DB.Exec(`INSERT INTO sessions(token_hash, user_id, expires_at) VALUES(?,?,?)`,
		tokenHash, userID, expires.UTC().Format(time.RFC3339))
	return err
}

// UserBySession returns the user for a live session; expired sessions are removed.
func (s *Store) UserBySession(tokenHash string) (User, error) {
	var u User
	var exp string
	err := s.DB.QueryRow(`SELECT u.id, u.email, u.name, u.password_hash, u.created_at, s.expires_at
		FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash=?`, tokenHash).
		Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.CreatedAt, &exp)
	if err != nil {
		return User{}, notFound(err)
	}
	t, perr := time.Parse(time.RFC3339, exp)
	if perr != nil || time.Now().After(t) {
		_ = s.DeleteSession(tokenHash)
		return User{}, ErrNotFound
	}
	return u, nil
}

func (s *Store) DeleteSession(tokenHash string) error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE token_hash=?`, tokenHash)
	return err
}

func (s *Store) PurgeExpiredSessions() error {
	_, err := s.DB.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
	return err
}

package httpapi

import (
	"net"
	"net/http"
	"strings"

	"restos/internal/auth"
	"restos/internal/db"
)

// dummyHash is compared against when the email is unknown, to keep login timing uniform.
var dummyHash, _ = auth.HashPassword("not-a-real-password")

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	if !s.Auth.AllowAttempt(clientIP(r)) {
		return apiError{http.StatusTooManyRequests, "too many attempts, try again later"}
	}
	var in struct{ Email, Password string }
	if err := decode(w, r, &in); err != nil {
		return err
	}
	u, err := s.Store.UserByEmail(in.Email)
	hash := dummyHash
	if err == nil {
		hash = u.PasswordHash
	}
	if !auth.CheckPassword(hash, in.Password) || err != nil {
		return apiError{http.StatusUnauthorized, "invalid email or password"}
	}
	if err := s.Auth.StartSession(w, u.ID); err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, u)
	return nil
}

func clientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// authConfig tells the sign-in pages which options are available.
func (s *Server) authConfig(w http.ResponseWriter, r *http.Request) error {
	writeJSON(w, http.StatusOK, map[string]bool{"registration": s.Registration})
	return nil
}

// register creates an account and signs it in; it only exists while RESTOS_REGISTRATION is on.
func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	if !s.Registration {
		return db.ErrNotFound
	}
	// Shares the login budget, so sign-up can't be used to hammer the server either.
	if !s.Auth.AllowAttempt(clientIP(r)) {
		return apiError{http.StatusTooManyRequests, "too many attempts, try again later"}
	}
	var in userInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	u, err := s.newUser(in)
	if err != nil {
		return err
	}
	if err := s.Auth.StartSession(w, u.ID); err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, u)
	return nil
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) error {
	s.Auth.EndSession(w, r)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	u, ok := s.Auth.CurrentUser(r)
	if !ok {
		return apiError{http.StatusUnauthorized, "unauthorized"}
	}
	writeJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) error {
	users, err := s.Store.ListUsers()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, users)
	return nil
}

type userInput struct{ Email, Name, Password string }

// newUser validates the input and creates the account (shared by admin "add user" and public sign-up).
func (s *Server) newUser(in userInput) (db.User, error) {
	email, err := cleanEmail(in.Email)
	if err != nil {
		return db.User{}, err
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 100 {
		return db.User{}, badRequest("name is required (max 100 characters)")
	}
	if len(in.Password) < 8 {
		return db.User{}, badRequest("password must be at least 8 characters")
	}
	if len(in.Password) > 72 {
		return db.User{}, badRequest("password must be at most 72 bytes")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return db.User{}, err
	}
	u, err := s.Store.CreateUser(email, in.Name, hash)
	if err == db.ErrEmailTaken {
		return db.User{}, apiError{http.StatusConflict, err.Error()}
	}
	return u, err
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) error {
	var in userInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	u, err := s.newUser(in)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, u)
	return nil
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	if me, _ := auth.UserFrom(r.Context()); me.ID == id {
		return badRequest("you cannot delete your own account")
	}
	if err := s.Store.DeleteUser(id); err == db.ErrLastUser {
		return badRequest(err.Error())
	} else if err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

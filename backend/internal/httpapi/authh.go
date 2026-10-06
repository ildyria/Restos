package httpapi

import (
	"net"
	"net/http"

	"restos/internal/auth"
	"restos/internal/db"
)

// dummyHash is compared against when the email is unknown, to keep login timing uniform.
var dummyHash, _ = auth.HashPassword("not-a-real-password")

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if !s.Auth.AllowAttempt(ip) {
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

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) error {
	var in struct{ Email, Name, Password string }
	if err := decode(w, r, &in); err != nil {
		return err
	}
	email, err := cleanEmail(in.Email)
	if err != nil {
		return err
	}
	if in.Name == "" {
		return badRequest("name is required")
	}
	if len(in.Password) < 8 {
		return badRequest("password must be at least 8 characters")
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return err
	}
	u, err := s.Store.CreateUser(email, in.Name, hash)
	if err == db.ErrEmailTaken {
		return apiError{http.StatusConflict, err.Error()}
	}
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

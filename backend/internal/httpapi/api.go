package httpapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"restos/internal/auth"
	"restos/internal/db"
	"restos/internal/media"
)

type Server struct {
	Store *db.Store
	Auth  *auth.Auth
	Media media.Store
}

type apiError struct {
	Status int
	Msg    string
}

func (e apiError) Error() string { return e.Msg }

func badRequest(msg string) error { return apiError{http.StatusBadRequest, msg} }

type handlerFunc func(w http.ResponseWriter, r *http.Request) error

// wrap turns an error-returning handler into an http.Handler with uniform JSON errors.
func wrap(h handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := h(w, r)
		if err == nil {
			return
		}
		var ae apiError
		switch {
		case errors.As(err, &ae):
			writeJSON(w, ae.Status, map[string]string{"error": ae.Msg})
		case errors.Is(err, db.ErrNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
		default:
			log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		}
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return badRequest("invalid JSON body")
	}
	return nil
}

func pathID(r *http.Request, name string) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		return 0, db.ErrNotFound
	}
	return id, nil
}

// Routes builds the full API handler (mounted by the caller under /api and /media).
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Public, read-only.
	mux.Handle("GET /api/restaurants", wrap(s.listRestaurants))
	mux.Handle("GET /api/restaurants/{slug}", wrap(s.getRestaurant))
	mux.Handle("GET /api/cuisines", wrap(s.listCuisines))
	mux.Handle("GET /media/{kind}/{file}", http.HandlerFunc(s.serveMedia))

	// Auth.
	mux.Handle("POST /api/auth/login", auth.CSRF(wrap(s.login)))
	mux.Handle("POST /api/auth/logout", auth.CSRF(wrap(s.logout)))
	mux.Handle("GET /api/auth/me", wrap(s.me))

	// Admin.
	admin := http.NewServeMux()
	admin.Handle("POST /api/admin/restaurants", wrap(s.createRestaurant))
	admin.Handle("GET /api/admin/restaurants/{id}", wrap(s.adminGetRestaurant))
	admin.Handle("PUT /api/admin/restaurants/{id}", wrap(s.updateRestaurant))
	admin.Handle("DELETE /api/admin/restaurants/{id}", wrap(s.deleteRestaurant))
	admin.Handle("PUT /api/admin/restaurants/{id}/cover", wrap(s.setCover))
	admin.Handle("PUT /api/admin/ranking", wrap(s.setRanking))
	admin.Handle("POST /api/admin/restaurants/{id}/visits", wrap(s.createVisit))
	admin.Handle("PUT /api/admin/visits/{id}", wrap(s.updateVisit))
	admin.Handle("DELETE /api/admin/visits/{id}", wrap(s.deleteVisit))
	admin.Handle("POST /api/admin/visits/{id}/dishes", wrap(s.createDish))
	admin.Handle("PUT /api/admin/dishes/{id}", wrap(s.updateDish))
	admin.Handle("DELETE /api/admin/dishes/{id}", wrap(s.deleteDish))
	admin.Handle("POST /api/admin/dishes/{id}/photos", wrap(s.uploadPhotos))
	admin.Handle("PUT /api/admin/photos/order", wrap(s.reorderPhotos))
	admin.Handle("DELETE /api/admin/photos/{id}", wrap(s.deletePhoto))
	admin.Handle("GET /api/admin/users", wrap(s.listUsers))
	admin.Handle("POST /api/admin/users", wrap(s.createUser))
	admin.Handle("DELETE /api/admin/users/{id}", wrap(s.deleteUser))
	mux.Handle("/api/admin/", auth.CSRF(s.Auth.Require(admin)))

	mux.Handle("/api/", wrap(func(http.ResponseWriter, *http.Request) error { return db.ErrNotFound }))
	return mux
}

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
	// Registration enables public sign-up at POST /api/auth/register.
	Registration bool
	// Fetch is the client for reading restaurant websites; nil uses site.SafeClient().
	Fetch *http.Client
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
		case errors.Is(err, db.ErrUnknownMedia):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "the website picture expired, enter the website again"})
		case errors.Is(err, db.ErrUnknownUser):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "scores refer to an unknown person"})
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
	mux.Handle("GET /api/people", wrap(s.listPeople))
	mux.Handle("GET /api/about", wrap(s.getAbout))
	mux.Handle("GET /media/{kind}/{file}", http.HandlerFunc(s.serveMedia))

	// Auth.
	mux.Handle("POST /api/auth/login", auth.CSRF(wrap(s.login)))
	mux.Handle("POST /api/auth/logout", auth.CSRF(wrap(s.logout)))
	mux.Handle("GET /api/auth/me", wrap(s.me))
	mux.Handle("GET /api/auth/config", wrap(s.authConfig))
	mux.Handle("POST /api/auth/register", auth.CSRF(wrap(s.register)))

	// Admin: every route in adminRoutes sits behind login and the CSRF check.
	admin := http.NewServeMux()
	for _, rt := range s.adminRoutes() {
		admin.Handle(rt.pattern, wrap(rt.handler))
	}
	mux.Handle("/api/admin/", auth.CSRF(s.Auth.Require(admin)))

	mux.Handle("/api/", wrap(func(http.ResponseWriter, *http.Request) error { return db.ErrNotFound }))
	return mux
}

type route struct {
	pattern string
	handler handlerFunc
}

// adminRoutes lists every admin endpoint. Routes registers them behind auth.Require, and
// TestAdminRoutesRequireLogin walks this same list, so a new admin route is covered automatically.
func (s *Server) adminRoutes() []route {
	return []route{
		{"POST /api/admin/restaurants", s.createRestaurant},
		{"GET /api/admin/restaurants/{id}", s.adminGetRestaurant},
		{"PUT /api/admin/restaurants/{id}", s.updateRestaurant},
		{"DELETE /api/admin/restaurants/{id}", s.deleteRestaurant},
		{"PUT /api/admin/restaurants/{id}/cover", s.setCover},
		{"POST /api/admin/restaurants/{id}/photo", s.uploadRestaurantPhoto},
		{"PUT /api/admin/ranking", s.setRanking},
		{"POST /api/admin/restaurants/{id}/visits", s.createVisit},
		{"PUT /api/admin/visits/{id}", s.updateVisit},
		{"DELETE /api/admin/visits/{id}", s.deleteVisit},
		{"POST /api/admin/visits/{id}/dishes", s.createDish},
		{"PUT /api/admin/dishes/{id}", s.updateDish},
		{"DELETE /api/admin/dishes/{id}", s.deleteDish},
		{"POST /api/admin/dishes/{id}/photos", s.uploadDishPhotos},
		{"POST /api/admin/visits/{id}/photos", s.uploadVisitPhotos},
		{"PUT /api/admin/photos/{id}", s.setPhotoDish},
		{"POST /api/admin/site-preview", s.sitePreview},
		{"PUT /api/admin/photos/order", s.reorderPhotos},
		{"DELETE /api/admin/photos/{id}", s.deletePhoto},
		{"GET /api/admin/about", s.adminGetAbout},
		{"PUT /api/admin/about", s.setAbout},
		{"POST /api/admin/about/preview", s.previewAbout},
		{"POST /api/admin/about/photo", s.uploadAboutPhoto},
		{"DELETE /api/admin/about/photo", s.deleteAboutPhoto},
		{"GET /api/admin/users", s.listUsers},
		{"POST /api/admin/users", s.createUser},
		{"DELETE /api/admin/users/{id}", s.deleteUser},
	}
}

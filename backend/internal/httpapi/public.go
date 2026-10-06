package httpapi

import (
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
)

func (s *Server) listRestaurants(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	list, err := s.Store.ListRestaurants(dbFilter(q.Get("sort"), q.Get("q"), q.Get("cuisine")))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

func (s *Server) getRestaurant(w http.ResponseWriter, r *http.Request) error {
	d, err := s.Store.RestaurantBySlug(r.PathValue("slug"))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, d)
	return nil
}

func (s *Server) listCuisines(w http.ResponseWriter, r *http.Request) error {
	c, err := s.Store.Cuisines()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, c)
	return nil
}

var mediaFile = regexp.MustCompile(`^[0-9a-f]{24}\.jpg$`)

func (s *Server) serveMedia(w http.ResponseWriter, r *http.Request) {
	kind, file := r.PathValue("kind"), r.PathValue("file")
	if (kind != "full" && kind != "thumb") || !mediaFile.MatchString(file) || strings.Contains(file, "/") {
		http.NotFound(w, r)
		return
	}
	// File keys are random and never reused, so the content is immutable.
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, filepath.Join(s.Media.Dir, kind, file))
}

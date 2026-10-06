package httpapi

import (
	"net/http"
	"strings"
	"time"

	"restos/internal/db"
	"restos/internal/media"
	"restos/internal/site"
)

const (
	maxPhotoBytes  = 20 << 20
	maxUploadBytes = 100 << 20
)

func dbFilter(sort, q, cuisine string) db.ListFilter {
	return db.ListFilter{Sort: sort, Query: q, Cuisine: cuisine}
}

func (s *Server) respondRestaurant(w http.ResponseWriter, status int, id int64) error {
	d, err := s.Store.RestaurantByID(id)
	if err != nil {
		return err
	}
	writeJSON(w, status, d)
	return nil
}

func (s *Server) adminGetRestaurant(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusOK, id)
}

// defaultFirstVisit is today's meal, used when a restaurant is created without a first visit.
func defaultFirstVisit(now time.Time) *db.VisitInput {
	meal := "dinner"
	if now.Hour() < 16 {
		meal = "lunch"
	}
	return &db.VisitInput{VisitedOn: now.Format("2006-01-02"), Meal: meal}
}

func (s *Server) createRestaurant(w http.ResponseWriter, r *http.Request) error {
	var in db.RestaurantInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := cleanRestaurant(&in); err != nil {
		return err
	}
	// Every restaurant starts with a visit, so photos can be added straight away.
	if in.FirstVisit == nil {
		in.FirstVisit = defaultFirstVisit(time.Now())
	}
	if err := cleanVisit(in.FirstVisit); err != nil {
		return err
	}
	id, err := s.Store.CreateRestaurant(in)
	if err != nil {
		return err
	}
	if err := s.applySiteImage(id, in.SiteImageKey); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusCreated, id)
}

func (s *Server) updateRestaurant(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in db.RestaurantInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	in.FirstVisit = nil // only meaningful on creation
	if err := cleanRestaurant(&in); err != nil {
		return err
	}
	if err := s.Store.UpdateRestaurant(id, in); err != nil {
		return err
	}
	if err := s.applySiteImage(id, in.SiteImageKey); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusOK, id)
}

func (s *Server) deleteRestaurant(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	keys, err := s.Store.DeleteRestaurant(id)
	if err != nil {
		return err
	}
	s.Media.Remove(keys)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) setCover(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	// {"photo_id": null} goes back to the automatic cover.
	var in struct {
		PhotoID *int64 `json:"photo_id"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := s.Store.SetCover(id, in.PhotoID); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusOK, id)
}

func (s *Server) setRanking(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, id := range in.IDs {
		if seen[id] {
			return badRequest("duplicate id in ranking")
		}
		seen[id] = true
	}
	if err := s.Store.SetRanking(in.IDs); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Visits

func (s *Server) createVisit(w http.ResponseWriter, r *http.Request) error {
	rid, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in db.VisitInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := cleanVisit(&in); err != nil {
		return err
	}
	if _, err := s.Store.RestaurantByID(rid); err != nil {
		return err
	}
	if _, err := s.Store.CreateVisit(rid, in); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusCreated, rid)
}

func (s *Server) updateVisit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in db.VisitInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := cleanVisit(&in); err != nil {
		return err
	}
	v, err := s.Store.VisitByID(id)
	if err != nil {
		return err
	}
	if err := s.Store.UpdateVisit(id, in); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusOK, v.RestaurantID)
}

func (s *Server) deleteVisit(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	v, err := s.Store.VisitByID(id)
	if err != nil {
		return err
	}
	keys, err := s.Store.DeleteVisit(id)
	if err != nil {
		return err
	}
	s.Media.Remove(keys)
	return s.respondRestaurant(w, http.StatusOK, v.RestaurantID)
}

// Dishes

func (s *Server) createDish(w http.ResponseWriter, r *http.Request) error {
	vid, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in db.DishInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := cleanDish(&in); err != nil {
		return err
	}
	v, err := s.Store.VisitByID(vid)
	if err != nil {
		return err
	}
	if _, err := s.Store.CreateDish(vid, in); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusCreated, v.RestaurantID)
}

func (s *Server) dishRestaurant(dishID int64) (int64, error) {
	d, err := s.Store.DishByID(dishID)
	if err != nil {
		return 0, err
	}
	v, err := s.Store.VisitByID(d.VisitID)
	return v.RestaurantID, err
}

func (s *Server) updateDish(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in db.DishInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := cleanDish(&in); err != nil {
		return err
	}
	rid, err := s.dishRestaurant(id)
	if err != nil {
		return err
	}
	if err := s.Store.UpdateDish(id, in); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusOK, rid)
}

func (s *Server) deleteDish(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	rid, err := s.dishRestaurant(id)
	if err != nil {
		return err
	}
	if err := s.Store.DeleteDish(id); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusOK, rid)
}

// Photos

// uploadVisitPhotos adds photos to a visit; their dish can be set afterwards.
func (s *Server) uploadVisitPhotos(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	v, err := s.Store.VisitByID(id)
	if err != nil {
		return err
	}
	return s.storeUploads(w, r, v.ID, nil, v.RestaurantID)
}

// uploadDishPhotos adds photos to a dish's visit, already captioned with that dish.
func (s *Server) uploadDishPhotos(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	d, err := s.Store.DishByID(id)
	if err != nil {
		return err
	}
	v, err := s.Store.VisitByID(d.VisitID)
	if err != nil {
		return err
	}
	return s.storeUploads(w, r, v.ID, &d.ID, v.RestaurantID)
}

func (s *Server) storeUploads(w http.ResponseWriter, r *http.Request, visitID int64, dishID *int64, rid int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		return badRequest("invalid or too large upload")
	}
	defer r.MultipartForm.RemoveAll()
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		return badRequest("no files provided (use the 'files' field)")
	}

	// Process every image first so a bad file doesn't leave a half-applied upload behind.
	var saved []media.Saved
	cleanup := func() {
		keys := make([]string, len(saved))
		for i, sv := range saved {
			keys[i] = sv.Key
		}
		s.Media.Remove(keys)
	}
	for _, fh := range files {
		if fh.Size > maxPhotoBytes {
			cleanup()
			return badRequest(fh.Filename + " is larger than 20 MB")
		}
		f, err := fh.Open()
		if err != nil {
			cleanup()
			return err
		}
		sv, err := s.Media.Save(f)
		f.Close()
		if err == media.ErrUnsupported {
			cleanup()
			return apiError{http.StatusUnsupportedMediaType, fh.Filename + ": " + err.Error()}
		}
		if err != nil {
			cleanup()
			return badRequest(fh.Filename + " could not be read as an image")
		}
		saved = append(saved, sv)
	}
	for i, sv := range saved {
		if _, err := s.Store.AddPhoto(visitID, dishID, sv.Key, sv.Width, sv.Height); err != nil {
			// Drop files whose rows were not created; earlier rows stay valid.
			s.Media.Remove(keysOf(saved[i:]))
			return err
		}
	}
	return s.respondRestaurant(w, http.StatusCreated, rid)
}

func keysOf(sv []media.Saved) []string {
	keys := make([]string, len(sv))
	for i, s := range sv {
		keys[i] = s.Key
	}
	return keys
}

func (s *Server) reorderPhotos(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		IDs []int64 `json:"ids"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := s.Store.ReorderPhotos(in.IDs); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) deletePhoto(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	rid, err := s.photoRestaurant(id)
	if err != nil {
		return err
	}
	keys, err := s.Store.DeletePhoto(id)
	if err != nil {
		return err
	}
	s.Media.Remove(keys)
	return s.respondRestaurant(w, http.StatusOK, rid)
}

func (s *Server) photoRestaurant(photoID int64) (int64, error) {
	p, err := s.Store.PhotoByID(photoID)
	return p.RestaurantID, err
}

// applySiteImage attaches a fetched website picture to a saved restaurant (nil: leave as is).
func (s *Server) applySiteImage(restaurantID int64, key *string) error {
	if key == nil {
		return nil
	}
	old, err := s.Store.SetSitePhoto(restaurantID, *key)
	if err != nil {
		return err
	}
	s.Media.Remove(old)
	return nil
}

// sitePreview reads a website as soon as its address is entered (before the restaurant is
// saved): it returns the site's name and downloads its og:image as a pending picture, which the
// restaurant claims on save via site_image_key.
func (s *Server) sitePreview(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		URL string `json:"url"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if _, err := site.ParseURL(in.URL); err != nil {
		return badRequest(err.Error())
	}
	// Pictures fetched for a restaurant that was never saved are dropped after a day.
	if stale, err := s.Store.PurgePendingMedia(time.Now().Add(-24 * time.Hour)); err == nil {
		s.Media.Remove(stale)
	}

	client := s.Fetch
	if client == nil {
		client = site.SafeClient()
	}
	page, err := site.Fetch(r.Context(), client, in.URL)
	if err != nil {
		return apiError{http.StatusUnprocessableEntity, "could not read the website: " + err.Error()}
	}
	out := struct {
		Title    string `json:"title"`
		ImageKey string `json:"image_key,omitempty"`
		Width    int    `json:"width,omitempty"`
		Height   int    `json:"height,omitempty"`
		Note     string `json:"note,omitempty"`
	}{Title: page.Title}
	if page.Image == "" {
		out.Note = "The website has no preview picture."
		writeJSON(w, http.StatusOK, out)
		return nil
	}
	body, err := site.Image(r.Context(), client, page.Image)
	if err != nil {
		out.Note = "The website's picture could not be downloaded."
		writeJSON(w, http.StatusOK, out)
		return nil
	}
	sv, err := s.Media.Save(body)
	body.Close()
	if err != nil {
		out.Note = "The website's picture is in a format we can't use."
		writeJSON(w, http.StatusOK, out)
		return nil
	}
	if err := s.Store.AddPendingMedia(sv.Key, sv.Width, sv.Height); err != nil {
		s.Media.Remove([]string{sv.Key})
		return err
	}
	out.ImageKey, out.Width, out.Height = sv.Key, sv.Width, sv.Height
	writeJSON(w, http.StatusOK, out)
	return nil
}

// setPhotoDish sets what a photo shows: {"dish": "Steak frites"}, or "" to clear it.
func (s *Server) setPhotoDish(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	var in struct {
		Dish string `json:"dish"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if len(strings.TrimSpace(in.Dish)) > 200 {
		return badRequest("dish name is too long (max 200 characters)")
	}
	rid, err := s.photoRestaurant(id)
	if err != nil {
		return err
	}
	if err := s.Store.SetPhotoDish(id, in.Dish); err != nil {
		return err
	}
	return s.respondRestaurant(w, http.StatusOK, rid)
}

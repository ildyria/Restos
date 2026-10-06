package httpapi

import (
	"net/http"

	"restos/internal/db"
	"restos/internal/media"
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

func (s *Server) createRestaurant(w http.ResponseWriter, r *http.Request) error {
	var in db.RestaurantInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := cleanRestaurant(&in); err != nil {
		return err
	}
	id, err := s.Store.CreateRestaurant(in)
	if err != nil {
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
	if err := cleanRestaurant(&in); err != nil {
		return err
	}
	if err := s.Store.UpdateRestaurant(id, in); err != nil {
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
	var in struct {
		PhotoID int64 `json:"photo_id"`
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
	keys, err := s.Store.DeleteDish(id)
	if err != nil {
		return err
	}
	s.Media.Remove(keys)
	return s.respondRestaurant(w, http.StatusOK, rid)
}

// Photos

func (s *Server) uploadPhotos(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r, "id")
	if err != nil {
		return err
	}
	rid, err := s.dishRestaurant(id)
	if err != nil {
		return err
	}
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
		if _, err := s.Store.AddPhoto(id, sv.Key, sv.Width, sv.Height); err != nil {
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
	p, err := s.Store.PhotoByID(id)
	if err != nil {
		return err
	}
	rid, err := s.dishRestaurant(p.DishID)
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

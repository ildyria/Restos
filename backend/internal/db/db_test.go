package db

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func f(v float64) *float64 { return &v }

// person creates a user to attach scores to.
func person(t *testing.T, s *Store, name string) int64 {
	t.Helper()
	u, err := s.CreateUser(name+"@example.com", name, "x")
	if err != nil {
		t.Fatal(err)
	}
	return u.ID
}

func TestRestaurantCRUDAndSlug(t *testing.T) {
	s := newStore(t)
	sam := person(t, s, "Sam")
	a, err := s.CreateRestaurant(RestaurantInput{Name: "Café Été", Scores: []ScoreInput{{UserID: sam, RatingFood: f(4), RatingValue: f(5)}}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := s.CreateRestaurant(RestaurantInput{Name: "Cafe Ete"})
	da, _ := s.RestaurantByID(a)
	db, _ := s.RestaurantByID(b)
	if da.Slug != "cafe-ete" || db.Slug != "cafe-ete-2" {
		t.Fatalf("slugs = %q, %q", da.Slug, db.Slug)
	}
	if da.Overall == nil || *da.Overall != 4.5 {
		t.Fatalf("overall = %v, want 4.5", da.Overall)
	}
	if db.Overall != nil {
		t.Fatal("unrated restaurant should have nil overall")
	}
	if err := s.UpdateRestaurant(a, RestaurantInput{Name: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	if d, _ := s.RestaurantByID(a); d.Slug != "cafe-ete" || d.Name != "Renamed" {
		t.Fatalf("slug must stay stable on rename: %+v", d.Restaurant)
	}
	if _, err := s.RestaurantBySlug("nope"); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestRankingAndSort(t *testing.T) {
	s := newStore(t)
	sam := person(t, s, "Sam")
	ids := map[string]int64{}
	for name, r := range map[string]float64{"a": 3, "b": 5, "c": 4} {
		id, _ := s.CreateRestaurant(RestaurantInput{Name: name, Scores: []ScoreInput{{UserID: sam, RatingFood: f(r)}}})
		ids[name] = id
	}
	names := func(sort string) string {
		l, err := s.ListRestaurants(ListFilter{Sort: sort})
		if err != nil {
			t.Fatal(err)
		}
		out := ""
		for _, r := range l {
			out += r.Name
		}
		return out
	}
	if got := names("rating"); got != "bca" {
		t.Fatalf("rating order = %s", got)
	}
	if err := s.SetRanking([]int64{ids["a"], ids["c"]}); err != nil {
		t.Fatal(err)
	}
	// ranked first in order, then unranked ("b") after.
	if got := names("rank"); got != "acb" {
		t.Fatalf("rank order = %s", got)
	}
	if err := s.SetRanking([]int64{ids["b"]}); err != nil {
		t.Fatal(err)
	}
	if got := names("rank"); got != "bca" {
		t.Fatalf("after re-rank = %s", got)
	}
	if err := s.SetRanking([]int64{9999}); err == nil {
		t.Fatal("expected error for unknown id")
	}
}

func TestCascadeReturnsPhotoKeysAndClearsCover(t *testing.T) {
	s := newStore(t)
	rid, _ := s.CreateRestaurant(RestaurantInput{Name: "R"})
	vid, err := s.CreateVisit(rid, VisitInput{VisitedOn: "2026-01-02", Meal: "dinner"})
	if err != nil {
		t.Fatal(err)
	}
	did, _ := s.CreateDish(vid, DishInput{Name: "Pasta"})
	p1, _ := s.AddPhoto(vid, &did, "k1", 10, 10)
	if _, err := s.AddPhoto(vid, &did, "k2", 10, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCover(rid, &p1.ID); err != nil {
		t.Fatal(err)
	}
	d, _ := s.RestaurantByID(rid)
	if d.CoverKey == nil || *d.CoverKey != "k1" {
		t.Fatalf("cover = %v", d.CoverKey)
	}
	if len(d.Visits) != 1 || len(d.Visits[0].Dishes) != 1 || len(d.Visits[0].Dishes[0].Photos) != 2 {
		t.Fatalf("unexpected tree: %+v", d.Visits)
	}

	keys, err := s.DeletePhoto(p1.ID)
	if err != nil || len(keys) != 1 || keys[0] != "k1" {
		t.Fatalf("keys=%v err=%v", keys, err)
	}
	d, _ = s.RestaurantByID(rid)
	if d.CoverPhotoID != nil {
		t.Fatal("cover_photo_id should be cleared")
	}
	if d.CoverKey == nil || *d.CoverKey != "k2" {
		t.Fatalf("cover should fall back to remaining photo, got %v", d.CoverKey)
	}

	keys, err = s.DeleteRestaurant(rid)
	if err != nil || len(keys) != 1 || keys[0] != "k2" {
		t.Fatalf("keys=%v err=%v", keys, err)
	}
	var n int
	s.DB.QueryRow(`SELECT COUNT(*) FROM visits`).Scan(&n)
	if n != 0 {
		t.Fatal("visits should cascade")
	}
}

func TestSetCoverRejectsForeignPhoto(t *testing.T) {
	s := newStore(t)
	r1, _ := s.CreateRestaurant(RestaurantInput{Name: "one"})
	r2, _ := s.CreateRestaurant(RestaurantInput{Name: "two"})
	v, _ := s.CreateVisit(r2, VisitInput{VisitedOn: "2026-01-02", Meal: "lunch"})
	p, _ := s.AddPhoto(v, nil, "k", 1, 1)
	if err := s.SetCover(r1, &p.ID); err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestUsersAndSessions(t *testing.T) {
	s := newStore(t)
	u, err := s.CreateUser(" Me@Example.com ", "Me", "hash")
	if err != nil || u.Email != "me@example.com" {
		t.Fatalf("u=%+v err=%v", u, err)
	}
	if _, err := s.CreateUser("me@example.com", "Dup", "h"); err != ErrEmailTaken {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
	if err := s.DeleteUser(u.ID); err != ErrLastUser {
		t.Fatalf("err = %v, want ErrLastUser", err)
	}

	if err := s.CreateSession("live", u.ID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got, err := s.UserBySession("live"); err != nil || got.ID != u.ID {
		t.Fatalf("live session: %v %v", got, err)
	}
	if err := s.CreateSession("old", u.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserBySession("old"); err != ErrNotFound {
		t.Fatalf("expired session err = %v", err)
	}

	u2, _ := s.CreateUser("b@example.com", "B", "h")
	if err := s.DeleteUser(u2.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteUser(9999); err != ErrNotFound && err != ErrLastUser {
		t.Fatalf("unexpected err %v", err)
	}
}

func TestPerPersonScores(t *testing.T) {
	s := newStore(t)
	sam, lea := person(t, s, "Sam"), person(t, s, "Lea")
	person(t, s, "Admin") // never scores, so not public
	yes, no := true, false
	id, err := s.CreateRestaurant(RestaurantInput{Name: "Duo", Latitude: f(51.05), Longitude: f(3.72), Scores: []ScoreInput{
		{UserID: sam, RatingFood: f(5), RatingService: f(4), WouldReturn: &yes},
		{UserID: lea, RatingFood: f(4), WouldReturn: &no},
		{UserID: 999}, // empty: dropped, even though the user does not exist
	}})
	if err != nil {
		t.Fatal(err)
	}
	d, _ := s.RestaurantByID(id)
	if *d.RatingFood != 4.5 || *d.RatingService != 4 || d.RatingAmbiance != nil {
		t.Fatalf("averages = %v %v %v", *d.RatingFood, *d.RatingService, d.RatingAmbiance)
	}
	if *d.Overall != 4.25 {
		t.Fatalf("overall = %v", *d.Overall)
	}
	if len(d.Scores) != 2 || d.Scores[0].Name != "Sam" || *d.Scores[0].Overall != 4.5 || !*d.Scores[0].WouldReturn || *d.Scores[1].WouldReturn {
		t.Fatalf("scores = %+v", d.Scores)
	}
	if *d.Latitude != 51.05 || *d.Longitude != 3.72 {
		t.Fatalf("location = %v,%v", *d.Latitude, *d.Longitude)
	}
	l, _ := s.ListRestaurants(ListFilter{})
	if len(l) != 1 || len(l[0].Scores) != 2 || *l[0].Overall != 4.25 {
		t.Fatalf("list = %+v", l)
	}
	if p, _ := s.People(); len(p) != 2 || p[0].Name != "Sam" || p[1].Name != "Lea" {
		t.Fatalf("people = %+v", p)
	}
	if err := s.UpdateRestaurant(id, RestaurantInput{Name: "Duo", Scores: []ScoreInput{{UserID: 999, RatingFood: f(3)}}}); !errors.Is(err, ErrUnknownUser) {
		t.Fatalf("unknown user err = %v", err)
	}
	if err := s.UpdateRestaurant(id, RestaurantInput{Name: "Duo", Scores: []ScoreInput{{UserID: lea, RatingFood: f(3)}}}); err != nil {
		t.Fatal(err)
	}
	if d, _ := s.RestaurantByID(id); len(d.Scores) != 1 || *d.RatingFood != 3 || d.Latitude != nil {
		t.Fatalf("after update = %+v", d.Restaurant)
	}
}

func TestPhotoCaptions(t *testing.T) {
	s := newStore(t)
	rid, _ := s.CreateRestaurant(RestaurantInput{Name: "R", FirstVisit: &VisitInput{VisitedOn: "2026-02-03", Meal: "lunch"}})
	d, _ := s.RestaurantByID(rid)
	if len(d.Visits) != 1 || d.Visits[0].VisitedOn != "2026-02-03" || d.Visits[0].Meal != "lunch" {
		t.Fatalf("first visit = %+v", d.Visits)
	}
	vid := d.Visits[0].ID
	other, _ := s.CreateVisit(rid, VisitInput{VisitedOn: "2026-01-01", Meal: "dinner"})
	otherDish, _ := s.CreateDish(other, DishInput{Name: "Tarte"})

	p1, _ := s.AddPhoto(vid, nil, "k1", 1, 1)
	p2, _ := s.AddPhoto(vid, nil, "k2", 1, 1)
	tree := func() Visit {
		d, _ := s.RestaurantByID(rid)
		for _, v := range d.Visits {
			if v.ID == vid {
				return v
			}
		}
		t.Fatal("visit missing")
		return Visit{}
	}
	if v := tree(); len(v.Photos) != 2 || len(v.Dishes) != 0 {
		t.Fatalf("uncaptioned = %+v", v)
	}

	// Captioning creates the dish once and reuses it case-insensitively, within the visit only.
	if err := s.SetPhotoDish(p1.ID, " Tarte "); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPhotoDish(p2.ID, "tarte"); err != nil {
		t.Fatal(err)
	}
	v := tree()
	if len(v.Dishes) != 1 || v.Dishes[0].Name != "Tarte" || v.Dishes[0].ID == otherDish || len(v.Dishes[0].Photos) != 2 || len(v.Photos) != 0 {
		t.Fatalf("captioned = %+v", v)
	}
	tarte := v.Dishes[0].ID

	// Renaming one photo keeps the dish (still used); renaming the last one removes it.
	s.SetPhotoDish(p1.ID, "Sole")
	s.SetPhotoDish(p2.ID, "Sole")
	if _, err := s.DishByID(tarte); err != ErrNotFound {
		t.Fatalf("unused caption dish should be gone, err = %v", err)
	}

	// A dish with notes survives losing its photos; clearing the caption unlinks the photo.
	v = tree()
	sole := v.Dishes[0].ID
	s.UpdateDish(sole, DishInput{Name: "Sole", Notes: "with beurre blanc"})
	s.SetPhotoDish(p1.ID, "")
	s.SetPhotoDish(p2.ID, "")
	if v = tree(); len(v.Dishes) != 1 || len(v.Photos) != 2 {
		t.Fatalf("after clearing = %+v", v)
	}

	// Deleting a dish keeps its photos.
	s.SetPhotoDish(p1.ID, "Sole")
	if err := s.DeleteDish(sole); err != nil {
		t.Fatal(err)
	}
	if v = tree(); len(v.Dishes) != 0 || len(v.Photos) != 2 {
		t.Fatalf("after dish delete = %+v", v)
	}
	if err := s.SetPhotoDish(9999, "x"); err != ErrNotFound {
		t.Fatalf("unknown photo err = %v", err)
	}
}

func TestSitePhoto(t *testing.T) {
	s := newStore(t)
	rid, _ := s.CreateRestaurant(RestaurantInput{Name: "R", FirstVisit: &VisitInput{VisitedOn: "2026-01-01", Meal: "dinner"}})
	d, _ := s.RestaurantByID(rid)
	visitPhoto, _ := s.AddPhoto(d.Visits[0].ID, nil, "visit", 1, 1)

	if _, err := s.SetSitePhoto(rid, "never-fetched"); err != ErrUnknownMedia {
		t.Fatalf("unknown key err = %v", err)
	}
	s.AddPendingMedia("og1", 1200, 630)
	if old, err := s.SetSitePhoto(rid, "og1"); err != nil || len(old) != 0 {
		t.Fatalf("claim = %v %v", old, err)
	}
	d, _ = s.RestaurantByID(rid)
	if d.SitePhoto == nil || d.SitePhoto.Key != "og1" || d.SitePhoto.VisitID != nil || *d.CoverKey != "og1" {
		t.Fatalf("site photo = %+v cover = %v (website picture is the default cover)", d.SitePhoto, d.CoverKey)
	}
	if len(d.Visits[0].Photos) != 1 {
		t.Fatal("website picture must not show up among visit photos")
	}
	if _, err := s.SetSitePhoto(rid, "og1"); err != nil {
		t.Fatalf("saving again with the same picture: %v", err)
	}
	if err := s.SetPhotoDish(d.SitePhoto.ID, "x"); err != ErrNotFound {
		t.Fatalf("captioning the website picture err = %v", err)
	}

	// Chosen as cover, then replaced: the cover follows the new picture.
	s.SetCover(rid, &d.SitePhoto.ID)
	s.AddPendingMedia("og2", 1, 1)
	if old, err := s.SetSitePhoto(rid, "og2"); err != nil || len(old) != 1 || old[0] != "og1" {
		t.Fatalf("replace = %v %v", old, err)
	}
	d, _ = s.RestaurantByID(rid)
	if d.CoverPhotoID == nil || *d.CoverPhotoID != d.SitePhoto.ID || *d.CoverKey != "og2" {
		t.Fatalf("cover after replace = %v %v", d.CoverPhotoID, d.CoverKey)
	}
	if _, err := s.SetSitePhoto(rid, "og2-again"); err != ErrUnknownMedia {
		t.Fatal("a claimed key can't be reused")
	}

	// Choosing a visit photo, then automatic again.
	s.SetCover(rid, &visitPhoto.ID)
	if d, _ = s.RestaurantByID(rid); *d.CoverKey != "visit" {
		t.Fatalf("chosen cover = %v", *d.CoverKey)
	}
	s.SetCover(rid, nil)
	if d, _ = s.RestaurantByID(rid); d.CoverPhotoID != nil || *d.CoverKey != "og2" {
		t.Fatalf("automatic cover = %v", *d.CoverKey)
	}

	// Removing the website picture falls back to the visit photo.
	if old, err := s.SetSitePhoto(rid, ""); err != nil || len(old) != 1 || old[0] != "og2" {
		t.Fatalf("remove = %v %v", old, err)
	}
	if d, _ = s.RestaurantByID(rid); d.SitePhoto != nil || *d.CoverKey != "visit" {
		t.Fatalf("after remove = %+v %v", d.SitePhoto, *d.CoverKey)
	}

	// Deleting the restaurant returns every file, website picture included.
	s.AddPendingMedia("og3", 1, 1)
	s.SetSitePhoto(rid, "og3")
	keys, _ := s.DeleteRestaurant(rid)
	if len(keys) != 2 {
		t.Fatalf("delete keys = %v", keys)
	}
}

func TestPurgePendingMedia(t *testing.T) {
	s := newStore(t)
	s.AddPendingMedia("old", 1, 1)
	s.DB.Exec(`UPDATE pending_media SET created_at = '2020-01-01T00:00:00Z' WHERE file_key = 'old'`)
	s.AddPendingMedia("fresh", 1, 1)
	keys, err := s.PurgePendingMedia(time.Now().Add(-24 * time.Hour))
	if err != nil || len(keys) != 1 || keys[0] != "old" {
		t.Fatalf("purge = %v %v", keys, err)
	}
}

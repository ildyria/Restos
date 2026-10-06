package db

import (
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

func TestRestaurantCRUDAndSlug(t *testing.T) {
	s := newStore(t)
	a, err := s.CreateRestaurant(RestaurantInput{Name: "Café Été", RatingFood: f(4), RatingValue: f(5)})
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
	ids := map[string]int64{}
	for name, r := range map[string]float64{"a": 3, "b": 5, "c": 4} {
		id, _ := s.CreateRestaurant(RestaurantInput{Name: name, RatingFood: f(r)})
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
	p1, _ := s.AddPhoto(did, "k1", 10, 10)
	if _, err := s.AddPhoto(did, "k2", 10, 10); err != nil {
		t.Fatal(err)
	}
	if err := s.SetCover(rid, p1.ID); err != nil {
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
	d, _ := s.CreateDish(v, DishInput{Name: "x"})
	p, _ := s.AddPhoto(d, "k", 1, 1)
	if err := s.SetCover(r1, p.ID); err != ErrNotFound {
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

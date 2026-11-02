package httpapi

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"restos/internal/auth"
	"restos/internal/db"
	"restos/internal/media"
)

type env struct {
	t      *testing.T
	srv    http.Handler
	media  string
	cookie *http.Cookie
	store  *db.Store
	server *Server
}

func newEnv(t *testing.T, opts ...func(*Server)) *env {
	t.Helper()
	dir := t.TempDir()
	st, err := db.Open(filepath.Join(dir, "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	hash, _ := auth.HashPassword("correct-horse")
	if _, err := st.CreateUser("admin@example.com", "Admin", hash); err != nil {
		t.Fatal(err)
	}
	s := &Server{Store: st, Auth: auth.New(st, false), Media: media.Store{Dir: filepath.Join(dir, "media")}}
	for _, o := range opts {
		o(s)
	}
	return &env{t: t, srv: s.Routes(), media: filepath.Join(dir, "media"), store: st, server: s}
}

func (e *env) do(method, path string, body any, csrf bool) *httptest.ResponseRecorder {
	var rd *bytes.Reader
	switch b := body.(type) {
	case nil:
		rd = bytes.NewReader(nil)
	case []byte:
		rd = bytes.NewReader(b)
	default:
		j, _ := json.Marshal(b)
		rd = bytes.NewReader(j)
	}
	return e.doReader(method, path, rd, "application/json", csrf)
}

func (e *env) doReader(method, path string, rd *bytes.Reader, ct string, csrf bool) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, rd)
	req.Header.Set("Content-Type", ct)
	if csrf {
		req.Header.Set("X-Requested-With", "restos")
	}
	if e.cookie != nil {
		req.AddCookie(e.cookie)
	}
	rec := httptest.NewRecorder()
	e.srv.ServeHTTP(rec, req)
	return rec
}

func (e *env) login() {
	rec := e.do("POST", "/api/auth/login", map[string]string{"email": "admin@example.com", "password": "correct-horse"}, true)
	if rec.Code != 200 {
		e.t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			e.cookie = c
		}
	}
	if e.cookie == nil {
		e.t.Fatal("no session cookie")
	}
}

func TestAuthFlow(t *testing.T) {
	e := newEnv(t)

	if rec := e.do("GET", "/api/auth/me", nil, false); rec.Code != 401 {
		t.Fatalf("me without session = %d", rec.Code)
	}
	if rec := e.do("POST", "/api/auth/login", map[string]string{"email": "admin@example.com", "password": "wrong"}, true); rec.Code != 401 {
		t.Fatalf("bad password = %d", rec.Code)
	}
	if rec := e.do("POST", "/api/auth/login", map[string]string{"email": "nobody@example.com", "password": "x"}, true); rec.Code != 401 {
		t.Fatalf("unknown user = %d", rec.Code)
	}
	if rec := e.do("POST", "/api/auth/login", map[string]string{"email": "admin@example.com", "password": "correct-horse"}, false); rec.Code != 403 {
		t.Fatalf("login without CSRF header = %d", rec.Code)
	}

	e.login()
	if rec := e.do("GET", "/api/auth/me", nil, false); rec.Code != 200 || strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("me = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("POST", "/api/auth/logout", nil, true); rec.Code != 204 {
		t.Fatalf("logout = %d", rec.Code)
	}
	if rec := e.do("GET", "/api/auth/me", nil, false); rec.Code != 401 {
		t.Fatalf("me after logout = %d", rec.Code)
	}
}

func TestAdminRequiresSessionAndCSRF(t *testing.T) {
	e := newEnv(t)
	body := map[string]any{"name": "X"}
	if rec := e.do("POST", "/api/admin/restaurants", body, true); rec.Code != 401 {
		t.Fatalf("no session = %d", rec.Code)
	}
	e.login()
	if rec := e.do("POST", "/api/admin/restaurants", body, false); rec.Code != 403 {
		t.Fatalf("no CSRF header = %d", rec.Code)
	}
	if rec := e.do("POST", "/api/admin/restaurants", body, true); rec.Code != 201 {
		t.Fatalf("create = %d %s", rec.Code, rec.Body)
	}
}

func TestValidation(t *testing.T) {
	e := newEnv(t)
	e.login()
	for name, body := range map[string]map[string]any{
		"empty name":   {"name": "  "},
		"bad rating":   {"name": "x", "scores": []map[string]any{{"user_id": 1, "rating_food": 4.3}}},
		"rating range": {"name": "x", "scores": []map[string]any{{"user_id": 1, "rating_food": 6}}},
		"rating zero":  {"name": "x", "scores": []map[string]any{{"user_id": 1, "rating_quiet": 0}}},
		"dup person":   {"name": "x", "scores": []map[string]any{{"user_id": 1, "rating_food": 3}, {"user_id": 1, "rating_food": 4}}},
		"no person":    {"name": "x", "scores": []map[string]any{{"user_id": 42, "rating_food": 3}}},
		"half coords":  {"name": "x", "latitude": 51},
		"bad coords":   {"name": "x", "latitude": 91, "longitude": 3},
		"bad price":    {"name": "x", "price_level": 9},
		"bad stars":    {"name": "x", "michelin_stars": 4},
		"bad website":  {"name": "x", "website": "javascript:alert(1)"},
	} {
		if rec := e.do("POST", "/api/admin/restaurants", body, true); rec.Code != 400 {
			t.Errorf("%s: code = %d", name, rec.Code)
		}
	}
	if rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "ok", "scores": []map[string]any{{"user_id": 1, "rating_food": 3.5, "would_return": true}}}, true); rec.Code != 201 {
		t.Fatalf("valid half-star = %d", rec.Code)
	}
	// The lowest rating is half a point, for every criterion (the database accepts it too).
	low := map[string]any{"user_id": 1, "rating_food": 0.5, "rating_service": 0.5, "rating_ambiance": 0.5, "rating_value": 0.5, "rating_quiet": 0.5}
	if rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "low", "scores": []map[string]any{low}}, true); rec.Code != 201 {
		t.Fatalf("half-point ratings = %d %s", rec.Code, rec.Body)
	}
}

func jpegBytes(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 120, 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func multipartBody(t *testing.T, files map[string][]byte) (*bytes.Reader, string) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for name, data := range files {
		fw, _ := mw.CreateFormFile("files", name)
		fw.Write(data)
	}
	mw.Close()
	return bytes.NewReader(buf.Bytes()), mw.FormDataContentType()
}

func TestFullFlowWithUpload(t *testing.T) {
	e := newEnv(t)
	e.login()

	var rest db.RestaurantDetail
	firstVisit := map[string]string{"visited_on": "2026-01-15", "meal": "lunch"}
	rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Chez Test", "cuisine": "French", "scores": []map[string]any{{"user_id": 1, "rating_food": 4.5}}, "first_visit": firstVisit}, true)
	json.Unmarshal(rec.Body.Bytes(), &rest)
	base := "/api/admin/restaurants/"

	if len(rest.Visits) != 1 || rest.Visits[0].VisitedOn != "2026-01-15" {
		t.Fatalf("first visit = %+v", rest.Visits)
	}
	vid := rest.Visits[0].ID

	rec = e.do("POST", "/api/admin/visits/"+itoa(vid)+"/dishes", map[string]any{"name": "Steak frites", "course": "main"}, true)
	if rec.Code != 201 {
		t.Fatalf("dish = %d %s", rec.Code, rec.Body)
	}
	rest = db.RestaurantDetail{}
	json.Unmarshal(rec.Body.Bytes(), &rest)
	did := rest.Visits[0].Dishes[0].ID

	// Upload a big image plus a non-image.
	rd, ct := multipartBody(t, map[string][]byte{"big.jpg": jpegBytes(3000, 1500)})
	rec = e.doReader("POST", "/api/admin/dishes/"+itoa(did)+"/photos", rd, ct, true)
	if rec.Code != 201 {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	rest = db.RestaurantDetail{}
	json.Unmarshal(rec.Body.Bytes(), &rest)
	photo := rest.Visits[0].Dishes[0].Photos[0]
	if photo.Width != 2000 || photo.Height != 1000 {
		t.Fatalf("photo not resized: %dx%d", photo.Width, photo.Height)
	}
	for _, kind := range []string{"full", "thumb"} {
		if _, err := os.Stat(filepath.Join(e.media, kind, photo.Key+".jpg")); err != nil {
			t.Fatalf("%s file missing: %v", kind, err)
		}
	}

	// Photos can go straight onto the visit, then be captioned with a dish by name.
	rd, ct = multipartBody(t, map[string][]byte{"plain.jpg": jpegBytes(40, 30)})
	if rec = e.doReader("POST", "/api/admin/visits/"+itoa(vid)+"/photos", rd, ct, true); rec.Code != 201 {
		t.Fatalf("visit upload = %d %s", rec.Code, rec.Body)
	}
	rest = db.RestaurantDetail{}
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if len(rest.Visits[0].Photos) != 1 || rest.Visits[0].Photos[0].DishID != nil {
		t.Fatalf("uncaptioned photos = %+v", rest.Visits[0].Photos)
	}
	loose := rest.Visits[0].Photos[0].ID
	if rec = e.do("PUT", "/api/admin/photos/"+itoa(loose), map[string]string{"dish": "steak FRITES"}, true); rec.Code != 200 {
		t.Fatalf("caption = %d %s", rec.Code, rec.Body)
	}
	rest = db.RestaurantDetail{}
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if len(rest.Visits[0].Photos) != 0 || len(rest.Visits[0].Dishes) != 1 || len(rest.Visits[0].Dishes[0].Photos) != 2 {
		t.Fatalf("caption should reuse the existing dish: %+v", rest.Visits[0])
	}
	if rec = e.do("PUT", "/api/admin/photos/"+itoa(loose), map[string]string{"dish": strings.Repeat("x", 201)}, true); rec.Code != 400 {
		t.Fatalf("long caption = %d", rec.Code)
	}
	// Deleting the dish keeps both photos on the visit.
	if rec = e.do("DELETE", "/api/admin/dishes/"+itoa(did), nil, true); rec.Code != 200 {
		t.Fatalf("delete dish = %d", rec.Code)
	}
	rest = db.RestaurantDetail{}
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if len(rest.Visits[0].Photos) != 2 || len(rest.Visits[0].Dishes) != 0 {
		t.Fatalf("after dish delete = %+v", rest.Visits[0])
	}

	rd, ct = multipartBody(t, map[string][]byte{"notes.txt": []byte("hello")})
	if rec = e.doReader("POST", "/api/admin/visits/"+itoa(vid)+"/photos", rd, ct, true); rec.Code != 415 && rec.Code != 400 {
		t.Fatalf("non-image upload = %d", rec.Code)
	}

	// Public detail (no cookie) shows cover; media is servable.
	pub := &env{t: t, srv: e.srv}
	rec = pub.do("GET", "/api/restaurants/chez-test", nil, false)
	rest = db.RestaurantDetail{}
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if rec.Code != 200 || rest.CoverKey == nil {
		t.Fatalf("public detail = %d cover=%v", rec.Code, rest.CoverKey)
	}
	if rec = pub.do("GET", "/media/thumb/"+photo.Key+".jpg", nil, false); rec.Code != 200 {
		t.Fatalf("media = %d", rec.Code)
	}
	if rec = pub.do("GET", "/media/full/..%2F..%2Fetc%2Fpasswd", nil, false); rec.Code != 404 {
		t.Fatalf("traversal = %d", rec.Code)
	}

	// Deleting the restaurant removes files from disk.
	if rec = e.do("DELETE", base+itoa(rest.ID), nil, true); rec.Code != 204 {
		t.Fatalf("delete = %d", rec.Code)
	}
	for _, kind := range []string{"full", "thumb"} {
		if _, err := os.Stat(filepath.Join(e.media, kind, photo.Key+".jpg")); !os.IsNotExist(err) {
			t.Fatalf("%s file should be gone (err=%v)", kind, err)
		}
	}
}

func TestUserManagementGuards(t *testing.T) {
	e := newEnv(t)
	e.login()
	var me db.User
	json.Unmarshal(e.do("GET", "/api/auth/me", nil, false).Body.Bytes(), &me)

	if rec := e.do("DELETE", "/api/admin/users/"+itoa(me.ID), nil, true); rec.Code != 400 {
		t.Fatalf("self delete = %d", rec.Code)
	}
	if rec := e.do("POST", "/api/admin/users", map[string]string{"email": "b@example.com", "name": "B", "password": "short"}, true); rec.Code != 400 {
		t.Fatalf("short password = %d", rec.Code)
	}
	if rec := e.do("POST", "/api/admin/users", map[string]string{"email": "b@example.com", "name": "B", "password": "long-enough-pw"}, true); rec.Code != 201 {
		t.Fatalf("create user = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("POST", "/api/admin/users", map[string]string{"email": "B@example.com", "name": "B2", "password": "long-enough-pw"}, true); rec.Code != 409 {
		t.Fatalf("duplicate = %d", rec.Code)
	}
}

func itoa(n int64) string {
	return strings.TrimSpace(strings.Replace(string(mustJSON(n)), "\n", "", -1))
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func TestRegistration(t *testing.T) {
	body := map[string]string{"email": "Lea@Example.com", "name": " Lea ", "password": "long-enough"}

	closed := newEnv(t)
	if rec := closed.do("GET", "/api/auth/config", nil, false); !strings.Contains(rec.Body.String(), `"registration":false`) {
		t.Fatalf("config = %s", rec.Body)
	}
	if rec := closed.do("POST", "/api/auth/register", body, true); rec.Code != 404 {
		t.Fatalf("register while disabled = %d", rec.Code)
	}

	e := newEnv(t, func(s *Server) { s.Registration = true })
	if rec := e.do("GET", "/api/auth/config", nil, false); !strings.Contains(rec.Body.String(), `"registration":true`) {
		t.Fatalf("config = %s", rec.Body)
	}
	if rec := e.do("POST", "/api/auth/register", body, false); rec.Code != 403 {
		t.Fatalf("register without CSRF header = %d", rec.Code)
	}
	for name, bad := range map[string]map[string]string{
		"short password": {"email": "a@example.com", "name": "A", "password": "short"},
		"long password":  {"email": "a@example.com", "name": "A", "password": strings.Repeat("x", 73)},
		"no name":        {"email": "a@example.com", "name": "  ", "password": "long-enough"},
		"bad email":      {"email": "nope", "name": "A", "password": "long-enough"},
		"taken email":    {"email": "admin@example.com", "name": "A", "password": "long-enough"},
	} {
		if rec := e.do("POST", "/api/auth/register", bad, true); rec.Code != 400 && rec.Code != 409 {
			t.Errorf("%s: code = %d", name, rec.Code)
		}
	}
	rec := e.do("POST", "/api/auth/register", body, true)
	if rec.Code != 201 || strings.Contains(rec.Body.String(), "password") {
		t.Fatalf("register = %d %s", rec.Code, rec.Body)
	}
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			e.cookie = c
		}
	}
	if rec := e.do("GET", "/api/auth/me", nil, false); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"name":"Lea"`) {
		t.Fatalf("me after register = %d %s", rec.Code, rec.Body)
	}
	// The new account is a working admin.
	if rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "By Lea"}, true); rec.Code != 201 {
		t.Fatalf("admin call as new user = %d", rec.Code)
	}
}

func TestFirstVisitOverride(t *testing.T) {
	e := newEnv(t)
	e.login()
	var rest db.RestaurantDetail
	rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Later", "first_visit": map[string]string{"visited_on": "2025-12-24", "meal": "dinner"}}, true)
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if rec.Code != 201 || len(rest.Visits) != 1 || rest.Visits[0].VisitedOn != "2025-12-24" || rest.Visits[0].Meal != "dinner" {
		t.Fatalf("first visit = %d %+v", rec.Code, rest.Visits)
	}
	if rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Bad", "first_visit": map[string]string{"visited_on": "soon", "meal": "dinner"}}, true); rec.Code != 400 {
		t.Fatalf("bad first visit = %d", rec.Code)
	}
	// Updating never adds visits.
	if rec := e.do("PUT", "/api/admin/restaurants/"+itoa(rest.ID), map[string]any{"name": "Later", "first_visit": map[string]string{"visited_on": "2025-01-01", "meal": "lunch"}}, true); rec.Code != 200 {
		t.Fatalf("update = %d", rec.Code)
	}
	if d, _ := e.get(rest.Slug); len(d.Visits) != 1 {
		t.Fatalf("update added a visit: %+v", d.Visits)
	}
}

// A restaurant can have no visits at all (we don't remember when we went), and get one later.
func TestRestaurantWithoutVisits(t *testing.T) {
	e := newEnv(t)
	e.login()
	var rest db.RestaurantDetail
	rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Long Ago", "scores": []map[string]any{{"user_id": 1, "rating_food": 4}}}, true)
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if rec.Code != 201 || len(rest.Visits) != 0 {
		t.Fatalf("create = %d %+v", rec.Code, rest.Visits)
	}
	if d, code := e.get(rest.Slug); code != 200 || len(d.Visits) != 0 {
		t.Fatalf("public detail = %d %+v", code, d.Visits)
	}
	var list []db.RestaurantSummary
	json.Unmarshal(e.do("GET", "/api/restaurants", nil, false).Body.Bytes(), &list)
	if len(list) != 1 || list[0].VisitCount != 0 || list[0].LastVisit != nil {
		t.Fatalf("public list = %+v", list)
	}
	if rec := e.do("POST", "/api/admin/restaurants/"+itoa(rest.ID)+"/visits", map[string]string{"visited_on": "2026-03-01", "meal": "dinner"}, true); rec.Code != 201 {
		t.Fatalf("add visit later = %d %s", rec.Code, rec.Body)
	}
	if d, _ := e.get(rest.Slug); len(d.Visits) != 1 {
		t.Fatalf("after adding a visit = %+v", d.Visits)
	}
}

func TestRestaurantPhoto(t *testing.T) {
	e := newEnv(t)
	e.login()
	var rest db.RestaurantDetail
	rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Chez Photo", "first_visit": map[string]string{"visited_on": "2026-01-15", "meal": "dinner"}}, true)
	json.Unmarshal(rec.Body.Bytes(), &rest)
	base := "/api/admin/restaurants/" + itoa(rest.ID)
	decode := func(rec *httptest.ResponseRecorder) db.RestaurantDetail {
		var d db.RestaurantDetail
		json.Unmarshal(rec.Body.Bytes(), &d)
		return d
	}
	exists := func(key string) bool {
		_, err := os.Stat(filepath.Join(e.media, "full", key+".jpg"))
		return err == nil
	}
	upload := func(path string) *httptest.ResponseRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", "front.jpg")
		fw.Write(jpegBytes(40, 30))
		mw.Close()
		return e.doReader("POST", path, bytes.NewReader(buf.Bytes()), mw.FormDataContentType(), true)
	}

	// A visit photo chosen as cover first.
	rd, ct := multipartBody(t, map[string][]byte{"dish.jpg": jpegBytes(40, 30)})
	rest = decode(e.doReader("POST", "/api/admin/visits/"+itoa(rest.Visits[0].ID)+"/photos", rd, ct, true))
	dish := rest.Visits[0].Photos[0]
	e.do("PUT", base+"/cover", map[string]any{"photo_id": dish.ID}, true)

	// The restaurant photo sits outside the visits and takes over the cover.
	rec = upload(base + "/photo")
	if rec.Code != 200 {
		t.Fatalf("upload = %d %s", rec.Code, rec.Body)
	}
	rest = decode(rec)
	first := rest.RestaurantPhoto
	if first == nil || first.Source != "restaurant" || first.VisitID != nil || !exists(first.Key) {
		t.Fatalf("restaurant photo = %+v", first)
	}
	if rest.CoverPhotoID == nil || *rest.CoverPhotoID != first.ID || len(rest.Visits[0].Photos) != 1 {
		t.Fatalf("after upload = %+v", rest)
	}
	if d, _ := e.get(rest.Slug); d.CoverKey == nil || *d.CoverKey != first.Key {
		t.Fatalf("public cover = %v", d.CoverKey)
	}
	if rec = e.do("PUT", "/api/admin/photos/"+itoa(first.ID), map[string]string{"dish": "Soup"}, true); rec.Code != 404 {
		t.Fatalf("caption restaurant photo = %d", rec.Code)
	}

	// A new upload replaces it, file included.
	rest = decode(upload(base + "/photo"))
	second := rest.RestaurantPhoto
	if second == nil || second.Key == first.Key || exists(first.Key) || !exists(second.Key) || *rest.CoverPhotoID != second.ID {
		t.Fatalf("replace = %+v", rest)
	}

	// Deleting it falls back to the automatic cover (the visit photo).
	rec = e.do("DELETE", "/api/admin/photos/"+itoa(second.ID), nil, true)
	rest = decode(rec)
	if rec.Code != 200 || rest.RestaurantPhoto != nil || rest.CoverPhotoID != nil || exists(second.Key) {
		t.Fatalf("delete = %d %+v", rec.Code, rest)
	}
	if d, _ := e.get(rest.Slug); d.CoverKey == nil || *d.CoverKey != dish.Key {
		t.Fatalf("cover after delete = %v", d.CoverKey)
	}

	if rec = upload("/api/admin/restaurants/9999/photo"); rec.Code != 404 {
		t.Fatalf("unknown restaurant = %d", rec.Code)
	}
}

func (e *env) get(slug string) (db.RestaurantDetail, int) {
	var d db.RestaurantDetail
	rec := e.do("GET", "/api/restaurants/"+slug, nil, false)
	json.Unmarshal(rec.Body.Bytes(), &d)
	return d, rec.Code
}

func TestSitePreview(t *testing.T) {
	img := jpegBytes(1200, 630)
	website := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			w.Write([]byte(`<head><meta property="og:site_name" content="Chez Test"><meta property="og:image" content="/og.jpg"></head>`))
		case "/og.jpg":
			w.Write(img)
		case "/bare":
			w.Write([]byte(`<title>Bare</title>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer website.Close()
	// The test website is on loopback, which the real client refuses; use a plain one.
	e := newEnv(t, func(s *Server) { s.Fetch = website.Client() })
	e.login()

	var prev struct {
		Title    string `json:"title"`
		ImageKey string `json:"image_key"`
		Width    int    `json:"width"`
		Note     string `json:"note"`
	}
	rec := e.do("POST", "/api/admin/site-preview", map[string]string{"url": website.URL + "/"}, true)
	json.Unmarshal(rec.Body.Bytes(), &prev)
	if rec.Code != 200 || prev.Title != "Chez Test" || prev.ImageKey == "" || prev.Width != 1200 {
		t.Fatalf("preview = %d %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(e.media, "thumb", prev.ImageKey+".jpg")); err != nil {
		t.Fatalf("preview image not stored: %v", err)
	}

	rec = e.do("POST", "/api/admin/site-preview", map[string]string{"url": website.URL + "/bare"}, true)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), `"title":"Bare"`) || strings.Contains(rec.Body.String(), "image_key") {
		t.Fatalf("no-image preview = %d %s", rec.Code, rec.Body)
	}
	for _, bad := range []string{"ftp://x", "", "not a url"} {
		if rec := e.do("POST", "/api/admin/site-preview", map[string]string{"url": bad}, true); rec.Code != 400 {
			t.Errorf("%q: code = %d", bad, rec.Code)
		}
	}
	if rec := e.do("POST", "/api/admin/site-preview", map[string]string{"url": website.URL + "/missing"}, true); rec.Code != 422 {
		t.Fatalf("404 website = %d", rec.Code)
	}

	// Saving the new restaurant claims the picture, which becomes its cover.
	var rest db.RestaurantDetail
	rec = e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Chez Test", "website": website.URL, "site_image_key": prev.ImageKey}, true)
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if rec.Code != 201 || rest.SitePhoto == nil || rest.SitePhoto.Key != prev.ImageKey || rest.CoverKey == nil || *rest.CoverKey != prev.ImageKey {
		t.Fatalf("create with site image = %d %s", rec.Code, rec.Body)
	}
	if rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Again", "site_image_key": prev.ImageKey}, true); rec.Code != 400 {
		t.Fatalf("reusing a claimed picture = %d", rec.Code)
	}
	if rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Bad", "site_image_key": "../../etc/passwd"}, true); rec.Code != 400 {
		t.Fatalf("bad key = %d", rec.Code)
	}
	// Removing it on update deletes the file.
	rec = e.do("PUT", "/api/admin/restaurants/"+itoa(rest.ID), map[string]any{"name": "Chez Test", "site_image_key": ""}, true)
	rest = db.RestaurantDetail{}
	json.Unmarshal(rec.Body.Bytes(), &rest)
	if rec.Code != 200 || rest.SitePhoto != nil {
		t.Fatalf("remove = %d %s", rec.Code, rec.Body)
	}
	if _, err := os.Stat(filepath.Join(e.media, "full", prev.ImageKey+".jpg")); !os.IsNotExist(err) {
		t.Fatalf("removed picture still on disk (err=%v)", err)
	}
	// Cover can be reset to automatic.
	if rec := e.do("PUT", "/api/admin/restaurants/"+itoa(rest.ID)+"/cover", map[string]any{"photo_id": nil}, true); rec.Code != 200 {
		t.Fatalf("automatic cover = %d", rec.Code)
	}
}

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
	"time"

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
	rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Chez Test", "cuisine": "French", "scores": []map[string]any{{"user_id": 1, "rating_food": 4.5}}}, true)
	json.Unmarshal(rec.Body.Bytes(), &rest)
	base := "/api/admin/restaurants/"

	// A restaurant starts with one visit (today, by default).
	if len(rest.Visits) != 1 || rest.Visits[0].VisitedOn != time.Now().Format("2006-01-02") {
		t.Fatalf("default first visit = %+v", rest.Visits)
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

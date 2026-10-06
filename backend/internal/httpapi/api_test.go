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
}

func newEnv(t *testing.T) *env {
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
	return &env{t: t, srv: s.Routes(), media: filepath.Join(dir, "media")}
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
		"bad rating":   {"name": "x", "rating_food": 4.3},
		"rating range": {"name": "x", "rating_food": 6},
		"bad price":    {"name": "x", "price_level": 9},
		"bad stars":    {"name": "x", "michelin_stars": 4},
		"bad website":  {"name": "x", "website": "javascript:alert(1)"},
	} {
		if rec := e.do("POST", "/api/admin/restaurants", body, true); rec.Code != 400 {
			t.Errorf("%s: code = %d", name, rec.Code)
		}
	}
	if rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "ok", "rating_food": 3.5}, true); rec.Code != 201 {
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
	rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Chez Test", "cuisine": "French", "rating_food": 4.5}, true)
	json.Unmarshal(rec.Body.Bytes(), &rest)
	base := "/api/admin/restaurants/"

	rec = e.do("POST", base+itoa(rest.ID)+"/visits", map[string]any{"visited_on": "2026-03-01", "meal": "lunch"}, true)
	if rec.Code != 201 {
		t.Fatalf("visit = %d %s", rec.Code, rec.Body)
	}
	json.Unmarshal(rec.Body.Bytes(), &rest)
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

	rd, ct = multipartBody(t, map[string][]byte{"notes.txt": []byte("hello")})
	if rec = e.doReader("POST", "/api/admin/dishes/"+itoa(did)+"/photos", rd, ct, true); rec.Code != 415 && rec.Code != 400 {
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

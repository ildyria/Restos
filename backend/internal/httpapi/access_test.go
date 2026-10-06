package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"restos/internal/auth"
)

// concrete turns a route pattern into a request line, e.g. "PUT /api/admin/photos/{id}" -> PUT, /api/admin/photos/1.
func concrete(pattern string) (method, path string) {
	method, path, _ = strings.Cut(pattern, " ")
	for _, p := range []string{"{id}", "{slug}"} {
		path = strings.ReplaceAll(path, p, "1")
	}
	return method, path
}

func isMutating(method string) bool {
	return method != http.MethodGet && method != http.MethodHead
}

// Every admin endpoint refuses requests without a valid session, whatever else they carry.
func TestAdminRoutesRequireLogin(t *testing.T) {
	e := newEnv(t)
	routes := e.server.adminRoutes()
	if len(routes) < 20 {
		t.Fatalf("only %d admin routes found; is adminRoutes still the single source?", len(routes))
	}

	// Something real behind the ids, so a missing check would show up as a 2xx/4xx other than 401.
	e.login()
	rec := e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Guarded"}, true)
	if rec.Code != 201 {
		t.Fatalf("setup: %d %s", rec.Code, rec.Body)
	}
	valid := e.cookie

	cases := map[string]*http.Cookie{
		"no cookie":     nil,
		"forged cookie": {Name: auth.CookieName, Value: "not-a-real-session-token"},
		"empty cookie":  {Name: auth.CookieName, Value: ""},
		"other cookie":  {Name: "session", Value: valid.Value},
	}
	for name, cookie := range cases {
		e.cookie = cookie
		for _, rt := range routes {
			method, path := concrete(rt.pattern)
			// With the CSRF header, so the 401 comes from the login check and not the CSRF check.
			rec := e.do(method, path, map[string]any{}, true)
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s: %s = %d, want 401", name, rt.pattern, rec.Code)
			}
		}
	}

	// Unknown admin paths don't reveal anything either.
	e.cookie = nil
	if rec := e.do("GET", "/api/admin/does-not-exist", nil, false); rec.Code != http.StatusUnauthorized {
		t.Errorf("unknown admin path = %d, want 401", rec.Code)
	}

	// Nothing changed: the restaurant made during setup is untouched.
	e.cookie = valid
	if rec := e.do("GET", "/api/restaurants/guarded", nil, false); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"name":"Guarded"`) {
		t.Fatalf("unauthenticated requests changed data: %d %s", rec.Code, rec.Body)
	}
}

// Logged in, but a mutating request without the CSRF header (e.g. a cross-site form post) is refused.
func TestAdminMutationsRequireCSRFHeader(t *testing.T) {
	e := newEnv(t)
	e.login()
	for _, rt := range e.server.adminRoutes() {
		method, path := concrete(rt.pattern)
		if !isMutating(method) {
			continue
		}
		if rec := e.do(method, path, map[string]any{}, false); rec.Code != http.StatusForbidden {
			t.Errorf("%s without CSRF header = %d, want 403", rt.pattern, rec.Code)
		}
	}
	// Right header, foreign origin.
	req := func() int {
		r, _ := http.NewRequest("POST", "/api/admin/restaurants", strings.NewReader(`{"name":"x"}`))
		r.Header.Set("X-Requested-With", "restos")
		r.Header.Set("Origin", "https://evil.example")
		r.AddCookie(e.cookie)
		w := httptest.NewRecorder()
		e.srv.ServeHTTP(w, r)
		return w.Code
	}
	if code := req(); code != http.StatusForbidden {
		t.Errorf("foreign origin = %d, want 403", code)
	}
}

// A session stops working after logout, once expired, and when its user is deleted.
func TestSessionsEnd(t *testing.T) {
	e := newEnv(t)
	probe := func() int { return e.do("GET", "/api/admin/users", nil, false).Code }

	e.login()
	if code := probe(); code != 200 {
		t.Fatalf("logged in = %d", code)
	}
	old := e.cookie
	e.do("POST", "/api/auth/logout", nil, true)
	e.cookie = old // replaying the cookie the browser was told to drop
	if code := probe(); code != http.StatusUnauthorized {
		t.Errorf("after logout = %d, want 401", code)
	}

	e.login()
	if _, err := e.store.DB.Exec(`UPDATE sessions SET expires_at = '2000-01-01T00:00:00Z'`); err != nil {
		t.Fatal(err)
	}
	if code := probe(); code != http.StatusUnauthorized {
		t.Errorf("expired session = %d, want 401", code)
	}

	// A second admin deletes the first: the first one's session dies with the account.
	e.login()
	first := e.cookie
	e.do("POST", "/api/admin/users", map[string]string{"email": "b@example.com", "name": "B", "password": "long-enough-pw"}, true)
	rec := e.do("POST", "/api/auth/login", map[string]string{"email": "b@example.com", "password": "long-enough-pw"}, true)
	for _, c := range rec.Result().Cookies() {
		if c.Name == auth.CookieName {
			e.cookie = c
		}
	}
	if rec := e.do("DELETE", "/api/admin/users/1", nil, true); rec.Code != 204 {
		t.Fatalf("delete first admin = %d %s", rec.Code, rec.Body)
	}
	e.cookie = first
	if code := probe(); code != http.StatusUnauthorized {
		t.Errorf("session of a deleted user = %d, want 401", code)
	}
}

// Public endpoints never expose account details (emails, password hashes).
func TestPublicResponsesHideAccounts(t *testing.T) {
	e := newEnv(t)
	e.login()
	e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Open", "scores": []map[string]any{{"user_id": 1, "rating_food": 4}}}, true)
	e.cookie = nil
	for _, path := range []string{"/api/restaurants", "/api/restaurants/open", "/api/people", "/api/about", "/api/auth/config"} {
		rec := e.do("GET", path, nil, false)
		if rec.Code != 200 {
			t.Errorf("%s = %d", path, rec.Code)
		}
		body := rec.Body.String()
		for _, secret := range []string{"admin@example.com", "password", "$2a$", "$2b$"} {
			if strings.Contains(body, secret) {
				t.Errorf("%s leaks %q: %s", path, secret, body)
			}
		}
	}
}

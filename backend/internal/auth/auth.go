package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"restos/internal/db"
)

const (
	CookieName  = "restos_session"
	SessionTTL  = 30 * 24 * time.Hour
	csrfHeader  = "X-Requested-With"
	csrfValue   = "restos"
	maxAttempts = 10
	attemptWin  = 15 * time.Minute
)

type ctxKey struct{}

type Auth struct {
	Store        *db.Store
	CookieSecure bool

	mu       sync.Mutex
	attempts map[string][]time.Time
}

func New(store *db.Store, cookieSecure bool) *Auth {
	return &Auth{Store: store, CookieSecure: cookieSecure, attempts: map[string][]time.Time{}}
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func hashToken(tok string) string {
	sum := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(sum[:])
}

// StartSession creates a session for the user and sets the cookie.
func (a *Auth) StartSession(w http.ResponseWriter, userID int64) error {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	tok := base64.RawURLEncoding.EncodeToString(raw)
	exp := time.Now().Add(SessionTTL)
	if err := a.Store.CreateSession(hashToken(tok), userID, exp); err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: tok, Path: "/", Expires: exp,
		HttpOnly: true, Secure: a.CookieSecure, SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (a *Auth) EndSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(CookieName); err == nil {
		_ = a.Store.DeleteSession(hashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: CookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: a.CookieSecure, SameSite: http.SameSiteLaxMode})
}

// CurrentUser returns the user attached to the request's session cookie, if any.
func (a *Auth) CurrentUser(r *http.Request) (db.User, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return db.User{}, false
	}
	u, err := a.Store.UserBySession(hashToken(c.Value))
	return u, err == nil
}

func UserFrom(ctx context.Context) (db.User, bool) {
	u, ok := ctx.Value(ctxKey{}).(db.User)
	return u, ok
}

// CSRF rejects mutating requests that lack the custom header or come from another origin.
func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if r.Header.Get(csrfHeader) != csrfValue {
				http.Error(w, `{"error":"missing CSRF header"}`, http.StatusForbidden)
				return
			}
			if o := r.Header.Get("Origin"); o != "" && o != "http://"+r.Host && o != "https://"+r.Host {
				http.Error(w, `{"error":"bad origin"}`, http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Require lets only authenticated requests through and stores the user in the context.
func (a *Auth) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := a.CurrentUser(r)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, u)))
	})
}

// AllowAttempt records a login attempt for key and reports whether it is within the limit.
func (a *Auth) AllowAttempt(key string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	cutoff := time.Now().Add(-attemptWin)
	kept := a.attempts[key][:0]
	for _, t := range a.attempts[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= maxAttempts {
		a.attempts[key] = kept
		return false
	}
	a.attempts[key] = append(kept, time.Now())
	return true
}

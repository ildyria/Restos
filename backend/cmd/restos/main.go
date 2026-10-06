package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/term"

	"restos/internal/auth"
	"restos/internal/config"
	"restos/internal/db"
	"restos/internal/httpapi"
	"restos/internal/media"
	"restos/internal/web"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cfg := config.Load()
	switch os.Args[1] {
	case "serve":
		fatal(serve(cfg))
	case "migrate":
		st, err := db.Open(cfg.DBPath())
		fatal(err)
		st.Close()
		fmt.Println("database is up to date")
	case "user":
		fatal(userCmd(cfg, os.Args[2:]))
	default:
		usage()
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `usage: restos <command>

  serve                                      start the web server
  migrate                                    apply database migrations
  user create --email E --name N             create an admin (prompts for password,
                                             or reads RESTOS_PASSWORD)
  user list                                  list admins
  user delete --email E                      delete an admin

environment: RESTOS_ADDR (:8080), RESTOS_DATA_DIR (./data), RESTOS_COOKIE_SECURE (true behind HTTPS)
`)
	os.Exit(2)
}

func fatal(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func serve(cfg config.Config) error {
	st, err := db.Open(cfg.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()
	_ = st.PurgeExpiredSessions()

	api := (&httpapi.Server{
		Store: st,
		Auth:  auth.New(st, cfg.CookieSecure),
		Media: media.Store{Dir: cfg.MediaDir()},
		// Every account is a full admin, so open sign-up is opt-in.
		Registration: cfg.Registration,
	}).Routes()

	mux := http.NewServeMux()
	mux.Handle("/api/", api)
	mux.Handle("/media/", api)
	mux.Handle("/", web.Handler())

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       2 * time.Minute, // photo uploads
		IdleTimeout:       2 * time.Minute,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	log.Printf("restos listening on %s (data: %s)", cfg.Addr, cfg.DataDir)

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func userCmd(cfg config.Config, args []string) error {
	if len(args) == 0 {
		usage()
	}
	st, err := db.Open(cfg.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()

	fs := flag.NewFlagSet("user "+args[0], flag.ExitOnError)
	email := fs.String("email", "", "email address")
	name := fs.String("name", "", "display name")
	_ = fs.Parse(args[1:])

	switch args[0] {
	case "create":
		if *email == "" || *name == "" {
			return errors.New("--email and --name are required")
		}
		pw, err := readPassword()
		if err != nil {
			return err
		}
		if len(pw) < 8 {
			return errors.New("password must be at least 8 characters")
		}
		hash, err := auth.HashPassword(pw)
		if err != nil {
			return err
		}
		u, err := st.CreateUser(*email, *name, hash)
		if err != nil {
			return err
		}
		fmt.Printf("created user %s <%s>\n", u.Name, u.Email)
	case "list":
		users, err := st.ListUsers()
		if err != nil {
			return err
		}
		for _, u := range users {
			fmt.Printf("%d\t%s\t%s\n", u.ID, u.Email, u.Name)
		}
	case "delete":
		u, err := st.UserByEmail(*email)
		if err != nil {
			return fmt.Errorf("user %q: %w", *email, err)
		}
		if err := st.DeleteUser(u.ID); err != nil {
			return err
		}
		fmt.Println("deleted", u.Email)
	default:
		usage()
	}
	return nil
}

func readPassword() (string, error) {
	if pw := os.Getenv("RESTOS_PASSWORD"); pw != "" {
		return pw, nil
	}
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("no terminal available: set RESTOS_PASSWORD")
	}
	fmt.Fprint(os.Stderr, "Password: ")
	a, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	fmt.Fprint(os.Stderr, "Confirm:  ")
	b, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	if string(a) != string(b) {
		return "", errors.New("passwords do not match")
	}
	return string(a), nil
}

package site

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParse(t *testing.T) {
	base, _ := url.Parse("https://chez.example/fr/accueil")
	doc := `<html><head><title> Chez  Marcel &amp; Fils </title>
		<meta content="/img/front.jpg?v=2&amp;w=1200" property="og:image">
		<META NAME='twitter:image' CONTENT='https://cdn.example/t.jpg'>
		<meta property="og:site_name" content="Brasserie Marcel"></head>`
	p := parse(doc, base)
	if p.Image != "https://chez.example/img/front.jpg?v=2&w=1200" {
		t.Fatalf("image = %q", p.Image)
	}
	if p.Title != "Brasserie Marcel" {
		t.Fatalf("title = %q", p.Title)
	}
	p = parse(`<title> Chez  Marcel &amp; Fils </title><meta name="twitter:image" content="t.png">`, base)
	if p.Title != "Chez Marcel & Fils" || p.Image != "https://chez.example/fr/t.png" {
		t.Fatalf("fallbacks = %+v", p)
	}
	if p := parse(`<p>nothing</p>`, base); p != (Page{}) {
		t.Fatalf("empty = %+v", p)
	}
}

func TestSafeClientRefusesPrivateAddresses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	if _, err := Fetch(context.Background(), SafeClient(), srv.URL); !errors.Is(err, ErrBlocked) {
		t.Fatalf("loopback err = %v", err)
	}
	for _, raw := range []string{"ftp://example.com", "/relative", "javascript:alert(1)"} {
		if _, err := Fetch(context.Background(), SafeClient(), raw); !errors.Is(err, ErrBadURL) {
			t.Errorf("%s: err = %v", raw, err)
		}
	}
	for ip, public := range map[string]bool{
		"10.1.2.3": false, "192.168.1.1": false, "169.254.169.254": false, "100.64.0.1": false,
		"::1": false, "fd00::1": false, "::ffff:127.0.0.1": false, "93.184.216.34": true, "2606:4700::1": true,
	} {
		if got := isPublic(net.ParseIP(ip)); got != public {
			t.Errorf("isPublic(%s) = %v", ip, got)
		}
	}
}

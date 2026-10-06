// Package site reads a restaurant's website for its picture (og:image) and name.
package site

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"syscall"
	"time"
)

const (
	maxPage      = 2 << 20 // the <head> is all we need
	maxImage     = 20 << 20
	maxRedirects = 5
	userAgent    = "Mozilla/5.0 (compatible; restos/1.0; +restaurant journal preview)"
)

var (
	ErrBadURL  = errors.New("website must be an http:// or https:// address")
	ErrBlocked = errors.New("website address is not public")
)

// Page is what we take from a website.
type Page struct {
	Image string // absolute URL of the og:image (or twitter:image), if any
	Title string // og:site_name, og:title or <title>
}

// SafeClient is an HTTP client for fetching user-supplied URLs from the server: it only
// connects to public addresses (checked at connect time, so redirects and DNS tricks are
// covered), ignores proxy settings, and limits time and redirects.
func SafeClient() *http.Client {
	dialer := &net.Dialer{Timeout: 5 * time.Second, Control: refusePrivate}
	return &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			Proxy:                 nil,
			DialContext:           dialer.DialContext,
			TLSHandshakeTimeout:   5 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
			MaxIdleConns:          4,
			IdleConnTimeout:       30 * time.Second,
		},
		CheckRedirect: checkRedirect,
	}
}

func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return errors.New("too many redirects")
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return ErrBadURL
	}
	return nil
}

func refusePrivate(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !isPublic(ip) {
		return ErrBlocked
	}
	return nil
}

var cgnat = &net.IPNet{IP: net.IPv4(100, 64, 0, 0), Mask: net.CIDRMask(10, 32)}

func isPublic(ip net.IP) bool {
	if v4 := ip.To4(); v4 != nil {
		ip = v4
	}
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || cgnat.Contains(ip))
}

// ParseURL checks that raw is an absolute http(s) URL.
func ParseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, ErrBadURL
	}
	return u, nil
}

func get(ctx context.Context, c *http.Client, u string, accept string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)
	res, err := c.Do(req)
	if err != nil {
		if errors.Is(err, ErrBlocked) {
			return nil, ErrBlocked
		}
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, fmt.Errorf("website answered %s", res.Status)
	}
	return res, nil
}

// Fetch reads the page at raw and returns its picture and title.
func Fetch(ctx context.Context, c *http.Client, raw string) (Page, error) {
	u, err := ParseURL(raw)
	if err != nil {
		return Page{}, err
	}
	res, err := get(ctx, c, u.String(), "text/html,application/xhtml+xml;q=0.9,*/*;q=0.5")
	if err != nil {
		return Page{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, maxPage))
	if err != nil {
		return Page{}, err
	}
	// Relative image paths resolve against the page we ended up on after redirects.
	return parse(string(body), res.Request.URL), nil
}

// Image downloads an image; the caller closes the body.
func Image(ctx context.Context, c *http.Client, imageURL string) (io.ReadCloser, error) {
	if _, err := ParseURL(imageURL); err != nil {
		return nil, err
	}
	res, err := get(ctx, c, imageURL, "image/avif,image/webp,image/png,image/jpeg,image/*;q=0.8")
	if err != nil {
		return nil, err
	}
	if res.ContentLength > maxImage {
		res.Body.Close()
		return nil, errors.New("website image is larger than 20 MB")
	}
	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(res.Body, maxImage), res.Body}, nil
}

var (
	metaTag  = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	attr     = regexp.MustCompile(`(?is)([a-z][a-z0-9:_-]*)\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s"'>]+))`)
	titleTag = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
)

func parse(doc string, base *url.URL) Page {
	meta := map[string]string{}
	for _, tag := range metaTag.FindAllString(doc, -1) {
		attrs := map[string]string{}
		for _, m := range attr.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(m[1])] = m[2] + m[3] + m[4]
		}
		key := strings.ToLower(attrs["property"])
		if key == "" {
			key = strings.ToLower(attrs["name"])
		}
		if _, seen := meta[key]; key != "" && !seen {
			meta[key] = strings.TrimSpace(html.UnescapeString(attrs["content"]))
		}
	}
	var p Page
	for _, k := range []string{"og:image:secure_url", "og:image", "og:image:url", "twitter:image", "twitter:image:src"} {
		if v := meta[k]; v != "" {
			if ref, err := url.Parse(v); err == nil {
				p.Image = base.ResolveReference(ref).String()
				break
			}
		}
	}
	for _, k := range []string{"og:site_name", "og:title"} {
		if v := meta[k]; v != "" {
			p.Title = v
			break
		}
	}
	if p.Title == "" {
		if m := titleTag.FindStringSubmatch(doc); m != nil {
			p.Title = strings.Join(strings.Fields(html.UnescapeString(m[1])), " ")
		}
	}
	if len(p.Title) > 200 {
		p.Title = p.Title[:200]
	}
	return p
}

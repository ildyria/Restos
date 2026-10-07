package httpapi

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

const (
	aboutTextKey  = "about_text"
	aboutPhotoKey = "about_photo"
	maxAboutChars = 20000
)

// DefaultAbout is the About page until an admin writes their own. {{names}} and {{restaurants}}
// are filled in when the page is shown.
const DefaultAbout = `We are {{names}}. This is our notebook of the {{restaurants}} we have eaten at: where we went, what we ordered, and whether we would go back.

Each of us scores every place on our own, from 1 to 5 in half steps, across five things: food, service, ambiance, value and how quiet the room is. The score you see in the list is the average of those.

The ranking is not sorted by score. It is our own order of preference, the list we would hand a friend who asked where to eat, so a place can sit above a higher-scoring one.
`

// Raw HTML in the source is dropped and dangerous link schemes are blanked (goldmark's defaults).
var markdown = goldmark.New(goldmark.WithExtensions(extension.GFM))

type aboutPage struct {
	HTML  string  `json:"html"`
	Photo *string `json:"photo"`
}

type adminAbout struct {
	Markdown        string  `json:"markdown"`
	IsDefault       bool    `json:"is_default"`
	DefaultMarkdown string  `json:"default_markdown"`
	Photo           *string `json:"photo"`
}

func (s *Server) aboutSource() (text string, isDefault bool, err error) {
	text, ok, err := s.Store.Setting(aboutTextKey)
	if err != nil || !ok {
		return DefaultAbout, true, err
	}
	return text, false, nil
}

func (s *Server) aboutPhoto() (*string, error) {
	key, ok, err := s.Store.Setting(aboutPhotoKey)
	if err != nil || !ok {
		return nil, err
	}
	return &key, nil
}

// renderAbout fills in the placeholders and turns the Markdown into HTML.
func (s *Server) renderAbout(src string) (string, error) {
	people, err := s.Store.People()
	if err != nil {
		return "", err
	}
	names := make([]string, len(people))
	for i, p := range people {
		names[i] = escapeMarkdown(p.Name)
	}
	n, err := s.Store.CountRestaurants()
	if err != nil {
		return "", err
	}
	restaurants := strconv.Itoa(n) + " restaurants"
	if n == 1 {
		restaurants = "1 restaurant"
	}
	who := joinNames(names)
	if who == "" {
		who = "the people behind this site"
	}
	src = strings.NewReplacer("{{names}}", who, "{{restaurants}}", restaurants).Replace(src)

	var buf bytes.Buffer
	if err := markdown.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// joinNames matches the frontend: "Sam", "Sam & Lea", "Sam, Lea & Kim".
func joinNames(names []string) string {
	if len(names) <= 1 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " & " + names[len(names)-1]
}

// escapeMarkdown keeps a name from being read as Markdown syntax.
func escapeMarkdown(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 128 && strings.ContainsRune("\\`*_{}[]()<>#+-.!|~&", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (s *Server) getAbout(w http.ResponseWriter, r *http.Request) error {
	src, _, err := s.aboutSource()
	if err != nil {
		return err
	}
	html, err := s.renderAbout(src)
	if err != nil {
		return err
	}
	photo, err := s.aboutPhoto()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, aboutPage{HTML: html, Photo: photo})
	return nil
}

func (s *Server) respondAdminAbout(w http.ResponseWriter) error {
	src, isDefault, err := s.aboutSource()
	if err != nil {
		return err
	}
	photo, err := s.aboutPhoto()
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, adminAbout{Markdown: src, IsDefault: isDefault, DefaultMarkdown: DefaultAbout, Photo: photo})
	return nil
}

func (s *Server) adminGetAbout(w http.ResponseWriter, r *http.Request) error {
	return s.respondAdminAbout(w)
}

func decodeAboutText(w http.ResponseWriter, r *http.Request) (string, error) {
	var in struct {
		Markdown string `json:"markdown"`
	}
	if err := decode(w, r, &in); err != nil {
		return "", err
	}
	if len([]rune(in.Markdown)) > maxAboutChars {
		return "", badRequest("the About text is limited to 20,000 characters")
	}
	return strings.TrimSpace(in.Markdown), nil
}

// setAbout saves the About text; an empty text (or the default itself) goes back to the default.
func (s *Server) setAbout(w http.ResponseWriter, r *http.Request) error {
	text, err := decodeAboutText(w, r)
	if err != nil {
		return err
	}
	if text == "" || text == strings.TrimSpace(DefaultAbout) {
		err = s.Store.DeleteSetting(aboutTextKey)
	} else {
		err = s.Store.SetSetting(aboutTextKey, text)
	}
	if err != nil {
		return err
	}
	return s.respondAdminAbout(w)
}

func (s *Server) previewAbout(w http.ResponseWriter, r *http.Request) error {
	text, err := decodeAboutText(w, r)
	if err != nil {
		return err
	}
	html, err := s.renderAbout(text)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]string{"html": html})
	return nil
}

// uploadAboutPhoto replaces the About page photo with the image in the "file" field.
func (s *Server) uploadAboutPhoto(w http.ResponseWriter, r *http.Request) error {
	sv, err := s.receivePhoto(w, r)
	if err != nil {
		return err
	}
	return s.replaceAboutPhoto(w, sv.Key)
}

func (s *Server) deleteAboutPhoto(w http.ResponseWriter, r *http.Request) error {
	return s.replaceAboutPhoto(w, "")
}

// replaceAboutPhoto points the About page at a new photo ("" for none) and deletes the old file.
func (s *Server) replaceAboutPhoto(w http.ResponseWriter, key string) error {
	old, err := s.aboutPhoto()
	if err != nil {
		return err
	}
	if key == "" {
		err = s.Store.DeleteSetting(aboutPhotoKey)
	} else {
		err = s.Store.SetSetting(aboutPhotoKey, key)
	}
	if err != nil {
		if key != "" {
			s.Media.Remove([]string{key})
		}
		return err
	}
	if old != nil {
		s.Media.Remove([]string{*old})
	}
	return s.respondAdminAbout(w)
}

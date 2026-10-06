package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func (e *env) about() aboutPage {
	e.t.Helper()
	rec := e.do("GET", "/api/about", nil, false)
	if rec.Code != 200 {
		e.t.Fatalf("about = %d %s", rec.Code, rec.Body)
	}
	var p aboutPage
	json.Unmarshal(rec.Body.Bytes(), &p)
	return p
}

func TestAboutText(t *testing.T) {
	e := newEnv(t)

	// Default text, with nobody having scored yet.
	if p := e.about(); !strings.Contains(p.HTML, "We are the people behind this site.") || !strings.Contains(p.HTML, "the 0 restaurants") || p.Photo != nil {
		t.Fatalf("default about = %+v", p)
	}

	e.login()
	e.do("POST", "/api/admin/restaurants", map[string]any{"name": "Chez Test", "scores": []map[string]any{{"user_id": 1, "rating_food": 4}}}, true)
	if p := e.about(); !strings.Contains(p.HTML, "We are Admin.") || !strings.Contains(p.HTML, "the 1 restaurant we") {
		t.Fatalf("filled-in about = %s", p.HTML)
	}

	if rec := e.do("PUT", "/api/admin/about", map[string]string{"markdown": "x"}, false); rec.Code != 403 {
		t.Fatalf("save without CSRF header = %d", rec.Code)
	}
	rec := e.do("PUT", "/api/admin/about", map[string]string{"markdown": "## Hi {{names}}\n\n<script>alert(1)</script>\n\n[x](javascript:alert(1))"}, true)
	var a adminAbout
	json.Unmarshal(rec.Body.Bytes(), &a)
	if rec.Code != 200 || a.IsDefault || !strings.HasPrefix(a.Markdown, "## Hi {{names}}") {
		t.Fatalf("save = %d %s", rec.Code, rec.Body)
	}
	p := e.about()
	if !strings.Contains(p.HTML, "<h2>Hi Admin</h2>") || strings.Contains(p.HTML, "<script") || strings.Contains(p.HTML, "javascript:") {
		t.Fatalf("custom about = %s", p.HTML)
	}

	// Saving an empty text goes back to the default.
	rec = e.do("PUT", "/api/admin/about", map[string]string{"markdown": "  "}, true)
	json.Unmarshal(rec.Body.Bytes(), &a)
	if !a.IsDefault || a.Markdown != DefaultAbout {
		t.Fatalf("reset = %+v", a)
	}

	rec = e.do("POST", "/api/admin/about/preview", map[string]string{"markdown": "*{{restaurants}}*"}, true)
	var preview aboutPage
	json.Unmarshal(rec.Body.Bytes(), &preview)
	if preview.HTML != "<p><em>1 restaurant</em></p>\n" {
		t.Fatalf("preview = %s", rec.Body)
	}
	if rec := e.do("PUT", "/api/admin/about", map[string]string{"markdown": strings.Repeat("a", maxAboutChars+1)}, true); rec.Code != 400 {
		t.Fatalf("too long = %d", rec.Code)
	}
}

func TestAboutPhoto(t *testing.T) {
	e := newEnv(t)
	e.login()

	upload := func() adminAbout {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, _ := mw.CreateFormFile("file", "us.jpg")
		fw.Write(jpegBytes(40, 30))
		mw.Close()
		rec := e.doReader("POST", "/api/admin/about/photo", bytes.NewReader(buf.Bytes()), mw.FormDataContentType(), true)
		if rec.Code != 200 {
			t.Fatalf("upload = %d %s", rec.Code, rec.Body)
		}
		var a adminAbout
		json.Unmarshal(rec.Body.Bytes(), &a)
		return a
	}
	exists := func(key string) bool {
		_, err := os.Stat(filepath.Join(e.media, "full", key+".jpg"))
		return err == nil
	}

	first := upload()
	if first.Photo == nil || !exists(*first.Photo) {
		t.Fatalf("first photo = %+v", first)
	}
	if p := e.about(); p.Photo == nil || *p.Photo != *first.Photo {
		t.Fatalf("public photo = %+v", p)
	}

	// Replacing the photo deletes the old file.
	second := upload()
	if *second.Photo == *first.Photo || exists(*first.Photo) || !exists(*second.Photo) {
		t.Fatalf("replace: first=%s second=%s", *first.Photo, *second.Photo)
	}

	rec := e.do("DELETE", "/api/admin/about/photo", nil, true)
	var a adminAbout
	json.Unmarshal(rec.Body.Bytes(), &a)
	if rec.Code != 200 || a.Photo != nil || exists(*second.Photo) || e.about().Photo != nil {
		t.Fatalf("delete = %d %s", rec.Code, rec.Body)
	}
}

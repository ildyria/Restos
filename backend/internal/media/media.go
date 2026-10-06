package media

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

const (
	fullMax  = 2000
	thumbMax = 480
)

var ErrUnsupported = errors.New("unsupported image: use JPEG, PNG or WebP")

type Store struct{ Dir string }

type Saved struct {
	Key           string
	Width, Height int
}

func (s Store) path(kind, key string) string { return filepath.Join(s.Dir, kind, key+".jpg") }

// Save decodes an uploaded image, applies EXIF orientation, and writes a full-size and a thumbnail
// JPEG. Re-encoding drops EXIF metadata such as GPS coordinates.
func (s Store) Save(r io.Reader) (Saved, error) {
	img, err := imaging.Decode(io.LimitReader(r, 25<<20), imaging.AutoOrientation(true))
	if err != nil {
		if errors.Is(err, image.ErrFormat) {
			return Saved{}, ErrUnsupported
		}
		return Saved{}, err
	}
	for _, k := range []string{"full", "thumb"} {
		if err := os.MkdirAll(filepath.Join(s.Dir, k), 0o755); err != nil {
			return Saved{}, err
		}
	}
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		return Saved{}, err
	}
	key := hex.EncodeToString(raw)

	full := fit(img, fullMax)
	if err := imaging.Save(full, s.path("full", key), imaging.JPEGQuality(85)); err != nil {
		return Saved{}, err
	}
	if err := imaging.Save(fit(img, thumbMax), s.path("thumb", key), imaging.JPEGQuality(80)); err != nil {
		_ = os.Remove(s.path("full", key))
		return Saved{}, err
	}
	b := full.Bounds()
	return Saved{Key: key, Width: b.Dx(), Height: b.Dy()}, nil
}

func (s Store) Remove(keys []string) {
	for _, k := range keys {
		_ = os.Remove(s.path("full", k))
		_ = os.Remove(s.path("thumb", k))
	}
}

// fit shrinks img so neither side exceeds max; it never upscales.
func fit(img image.Image, max int) image.Image {
	b := img.Bounds()
	if b.Dx() <= max && b.Dy() <= max {
		return img
	}
	return imaging.Fit(img, max, max, imaging.Lanczos)
}

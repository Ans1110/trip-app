package album

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp" // register webp decoder for image.Decode
)

// ThumbnailSize scales so the longest side matches Px; aspect ratio preserved.
type ThumbnailSize struct {
	Name string
	Px   int
}

var (
	ThumbSmall  = ThumbnailSize{Name: "small", Px: 400}
	ThumbMedium = ThumbnailSize{Name: "medium", Px: 1024}
)

type ThumbnailGenerator interface {
	Generate(ctx context.Context, src []byte, srcMime string, size ThumbnailSize) (data []byte, mime string, err error)
}

var ErrUnsupportedMime = errors.New("album: unsupported image mime")

type imagingThumbnailer struct{}

func NewThumbnailGenerator() ThumbnailGenerator { return &imagingThumbnailer{} }

func (t *imagingThumbnailer) Generate(ctx context.Context, src []byte, srcMime string, size ThumbnailSize) ([]byte, string, error) {
	if !supportedThumbMime(srcMime) {
		return nil, "", ErrUnsupportedMime
	}
	if ctx.Err() != nil {
		return nil, "", ctx.Err()
	}
	// AutoOrientation applies the EXIF Orientation tag before scaling so
	// portrait iPhone shots (Orientation=6) come out upright rather than sideways.
	img, err := imaging.Decode(bytes.NewReader(src), imaging.AutoOrientation(true))
	if err != nil {
		return nil, "", fmt.Errorf("album: decode: %w", err)
	}
	if ctx.Err() != nil {
		return nil, "", ctx.Err()
	}
	// Fit preserves aspect ratio, matches longest-side ≤ Px, and never upscales
	// (Clone returns identity when the source already fits the box).
	scaled := imaging.Fit(img, size.Px, size.Px, imaging.Lanczos)
	if ctx.Err() != nil {
		return nil, "", ctx.Err()
	}

	var buf bytes.Buffer
	if err := imaging.Encode(&buf, scaled, imaging.JPEG, imaging.JPEGQuality(82)); err != nil {
		return nil, "", fmt.Errorf("album: jpeg encode: %w", err)
	}
	return buf.Bytes(), "image/jpeg", nil
}

func supportedThumbMime(mime string) bool {
	switch mime {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

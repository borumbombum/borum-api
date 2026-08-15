// Package imgconv converts PNG and JPEG images to WebP in memory using a
// pure-Go encoder (github.com/KarpelesLab/gowebp): no CGO, no libwebp.
//
// This is the web-facing core of the img2webp CLI refactored for this
// project: it decodes from and encodes to streams instead of files, and the
// mode/quality/method knobs are passed in as options rather than CLI flags.
package imgconv

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/KarpelesLab/gowebp"
)

// Mode selects how the source is encoded.
type Mode int

const (
	// ModeAuto tries both lossless and lossy and keeps the smaller output.
	ModeAuto Mode = iota
	// ModeLossless forces lossless (VP8L) encoding.
	ModeLossless
	// ModeLossy forces lossy (VP8) encoding at the given quality.
	ModeLossy
)

// DefaultQuality is the lossy quality used when Options.Quality is zero.
const DefaultQuality float32 = 80

// DefaultMethod is the lossy speed/quality trade-off used when
// Options.Method is zero.
const DefaultMethod = 4

// Options controls how the source image is encoded.
type Options struct {
	Quality float32 // lossy quality in [0,100]; default 80
	Method  int     // lossy effort in [0,6]; default 4
	Mode    Mode
}

// Result is the encoded WebP plus a label describing which encoding won
// ("lossy" or "lossless").
type Result struct {
	Data  []byte
	Label string
}

// Convert decodes an image from r (restricted to PNG and JPEG, like the
// original tool) and encodes it as WebP according to o.
func Convert(r io.Reader, o Options) (Result, error) {
	img, format, err := image.Decode(r)
	if err != nil {
		return Result{}, fmt.Errorf("not a valid image: %w", err)
	}
	if format != "png" && format != "jpeg" {
		return Result{}, fmt.Errorf("unsupported format %s, expected PNG or JPEG", format)
	}
	return bestWebP(img, o)
}

// bestWebP encodes img according to the mode and returns the bytes plus a
// label for reporting.
func bestWebP(img image.Image, o Options) (Result, error) {
	q := o.Quality
	if q == 0 {
		q = DefaultQuality
	}
	m := o.Method
	if m == 0 {
		m = DefaultMethod
	}

	if o.Mode == ModeLossless {
		var buf bytes.Buffer
		if err := gowebp.Encode(&buf, img, nil); err != nil {
			return Result{}, err
		}
		return Result{Data: buf.Bytes(), Label: "lossless"}, nil
	}
	if o.Mode == ModeLossy {
		var buf bytes.Buffer
		if err := gowebp.Encode(&buf, img, &gowebp.Options{Lossy: true, Quality: q, Method: m}); err != nil {
			return Result{}, err
		}
		return Result{Data: buf.Bytes(), Label: "lossy"}, nil
	}

	// ModeAuto: try both, keep the smaller.
	var llBuf, lyBuf bytes.Buffer
	errLL := gowebp.Encode(&llBuf, img, nil)
	errLY := gowebp.Encode(&lyBuf, img, &gowebp.Options{Lossy: true, Quality: q, Method: m})

	switch {
	case errLL != nil && errLY != nil:
		return Result{}, errLL
	case errLL != nil:
		return Result{Data: lyBuf.Bytes(), Label: "lossy"}, nil
	case errLY != nil:
		return Result{Data: llBuf.Bytes(), Label: "lossless"}, nil
	case lyBuf.Len() < llBuf.Len():
		return Result{Data: lyBuf.Bytes(), Label: "lossy"}, nil
	default:
		return Result{Data: llBuf.Bytes(), Label: "lossless"}, nil
	}
}

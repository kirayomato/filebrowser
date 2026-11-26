package exifmeta

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"image"
	"image/png"
	"io"
)

const (
	// stealthMagic starts the payload NovelAI hides in the alpha LSBs.
	stealthMagic = "stealth_pngcomp"
	// stealthSizeLimit skips the full PNG decode for files too large to be
	// worth the cost; such images never carry this payload in practice.
	stealthSizeLimit = 64 << 20
	// stealthMaxBytes caps the inflated payload.
	stealthMaxBytes = 8 << 20
)

// stealthPayload decodes the column-major alpha LSB stream of a PNG and
// returns the inflated JSON payload, or nil when the magic does not match.
func stealthPayload(f Source) []byte {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil
	}
	img, err := png.Decode(f)
	if err != nil {
		return nil
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil
	}
	alpha, ok := alphaAt(img)
	if !ok {
		return nil
	}

	pos := 0
	readBytes := func(n int) ([]byte, bool) {
		if n <= 0 || pos+n*8 > w*h {
			return nil, false
		}
		out := make([]byte, n)
		for i := range out {
			var v byte
			for j := 0; j < 8; j++ {
				x, y := pos/h, pos%h // column-major
				pos++
				v |= (alpha(bounds.Min.X+x, bounds.Min.Y+y) & 1) << (7 - uint(j))
			}
			out[i] = v
		}
		return out, true
	}

	magic, ok := readBytes(len(stealthMagic))
	if !ok || string(magic) != stealthMagic {
		return nil
	}
	rawLen, ok := readBytes(4)
	if !ok {
		return nil
	}
	bits := binary.BigEndian.Uint32(rawLen)
	if bits == 0 || bits%8 != 0 || bits/8 > stealthMaxBytes {
		return nil
	}
	data, ok := readBytes(int(bits / 8))
	if !ok {
		return nil
	}

	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	defer zr.Close()
	payload, err := io.ReadAll(io.LimitReader(zr, stealthMaxBytes))
	if err != nil {
		return nil
	}
	return payload
}

// alphaAt returns accessors for the alpha byte of the common PNG layouts.
func alphaAt(img image.Image) (func(x, y int) byte, bool) {
	switch m := img.(type) {
	case *image.NRGBA:
		return func(x, y int) byte { return m.Pix[m.PixOffset(x, y)+3] }, true
	case *image.RGBA:
		return func(x, y int) byte { return m.Pix[m.PixOffset(x, y)+3] }, true
	case *image.NRGBA64:
		return func(x, y int) byte { return m.Pix[m.PixOffset(x, y)+7] }, true
	case *image.RGBA64:
		return func(x, y int) byte { return m.Pix[m.PixOffset(x, y)+7] }, true
	}
	return nil, false
}

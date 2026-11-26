package exifmeta

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"io"
	"strings"
)

// scanLimit caps a single chunk/segment read so a malformed file cannot force
// a huge allocation.
const scanLimit = 8 << 20

// xmpJPEGIdent marks an APP1 segment carrying an XMP packet.
const xmpJPEGIdent = "http://ns.adobe.com/xap/1.0/\x00"

type imageFormat int

const (
	fmtUnknown imageFormat = iota
	fmtJPEG
	fmtPNG
	fmtWEBP
)

func (f imageFormat) String() string {
	switch f {
	case fmtJPEG:
		return "jpeg"
	case fmtPNG:
		return "png"
	case fmtWEBP:
		return "webp"
	}
	return "unknown"
}

// sniff identifies the format from the file header, ignoring the extension.
func sniff(head []byte) imageFormat {
	switch {
	case len(head) >= 3 && head[0] == 0xFF && head[1] == 0xD8 && head[2] == 0xFF:
		return fmtJPEG
	case len(head) >= 8 && string(head[:8]) == "\x89PNG\r\n\x1a\n":
		return fmtPNG
	case len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP":
		return fmtWEBP
	}
	return fmtUnknown
}

// jpegChunks walks JPEG segments and returns the Exif TIFF payload from APP1.
func jpegChunks(f io.ReadSeeker, size int64) chunks {
	var c chunks
	off := int64(2) // SOI
	for off+4 <= size {
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return c
		}
		var marker [2]byte
		if _, err := io.ReadFull(f, marker[:]); err != nil {
			return c
		}
		if marker[0] != 0xFF {
			return c
		}
		// Standalone markers carry no length.
		if marker[1] == 0x01 || (marker[1] >= 0xD0 && marker[1] <= 0xD8) {
			off += 2
			continue
		}
		// Start of scan / end of image: no metadata past this point.
		if marker[1] == 0xDA || marker[1] == 0xD9 {
			return c
		}
		var l [2]byte
		if _, err := io.ReadFull(f, l[:]); err != nil {
			return c
		}
		segLen := int64(binary.BigEndian.Uint16(l[:]))
		if segLen < 2 {
			return c
		}
		if marker[1] == 0xE1 && segLen-2 >= 6 && segLen-2 <= scanLimit {
			buf := make([]byte, segLen-2)
			if _, err := io.ReadFull(f, buf); err != nil {
				return c
			}
			if c.tiff == nil && string(buf[:6]) == "Exif\x00\x00" {
				c.tiff = buf[6:]
			} else if c.xmp == nil && strings.HasPrefix(string(buf), xmpJPEGIdent) {
				c.xmp = buf[len(xmpJPEGIdent):]
			}
		}
		off += 2 + segLen
	}
	return c
}

// pngChunks walks PNG chunks and collects eXIf data plus tEXt/iTXt entries.
func pngChunks(f io.ReadSeeker, size int64) chunks {
	var c chunks
	off := int64(8) // signature
	for off+8 <= size {
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return c
		}
		var hdr [8]byte
		if _, err := io.ReadFull(f, hdr[:]); err != nil {
			return c
		}
		length := int64(binary.BigEndian.Uint32(hdr[:4]))
		typ := string(hdr[4:8])
		if length < 0 || off+8+length+4 > size {
			return c
		}
		switch typ {
		case "IEND":
			return c
		case "eXIf":
			if c.tiff == nil && length > 0 && length <= scanLimit {
				buf := make([]byte, length)
				if _, err := io.ReadFull(f, buf); err == nil {
					c.tiff = buf
				}
			}
		case "tEXt", "iTXt":
			if length > 0 && length <= scanLimit {
				buf := make([]byte, length)
				if _, err := io.ReadFull(f, buf); err == nil {
					key, value, ok := parseTextChunk(typ, buf)
					if ok {
						// XMP travels in an iTXt block in PNG.
						if strings.EqualFold(key, "XML:com.adobe.xmp") {
							if c.xmp == nil {
								c.xmp = []byte(value)
							}
						} else {
							c.texts = append(c.texts, textChunk{
								source: "png-" + typ,
								key:    key,
								value:  value,
							})
						}
					}
				}
			}
		}
		off += 8 + length + 4 // length + type + data + CRC
	}
	return c
}

// parseTextChunk decodes a PNG tEXt or iTXt payload.
func parseTextChunk(typ string, b []byte) (key, value string, ok bool) {
	i := bytes.IndexByte(b, 0)
	if i < 0 {
		return "", "", false
	}
	key = string(b[:i])
	rest := b[i+1:]

	if typ == "tEXt" {
		return key, string(rest), true
	}

	// iTXt: compression flag, compression method, language tag,
	// translated keyword, then text; all separated by NUL.
	if len(rest) < 2 {
		return "", "", false
	}
	compressed := rest[0] == 1
	rest = rest[2:]
	for i := 0; i < 2; i++ {
		j := bytes.IndexByte(rest, 0)
		if j < 0 {
			return "", "", false
		}
		rest = rest[j+1:]
	}
	if !compressed {
		return key, string(rest), true
	}
	zr, err := zlib.NewReader(bytes.NewReader(rest))
	if err != nil {
		return "", "", false
	}
	defer zr.Close()
	data, err := io.ReadAll(io.LimitReader(zr, scanLimit))
	if err != nil {
		return "", "", false
	}
	return key, string(data), true
}

// webpChunks walks RIFF chunks and returns the EXIF payload.
func webpChunks(f io.ReadSeeker, size int64) chunks {
	var c chunks
	off := int64(12) // RIFF header
	for off+8 <= size {
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return c
		}
		var hdr [8]byte
		if _, err := io.ReadFull(f, hdr[:]); err != nil {
			return c
		}
		id := string(hdr[:4])
		n := int64(binary.LittleEndian.Uint32(hdr[4:8]))
		if n < 0 || off+8+n > size {
			return c
		}
		if id == "EXIF" && c.tiff == nil && n > 0 && n <= scanLimit {
			buf := make([]byte, n)
			if _, err := io.ReadFull(f, buf); err == nil {
				if len(buf) > 6 && string(buf[:6]) == "Exif\x00\x00" {
					buf = buf[6:]
				}
				c.tiff = buf
			}
		}
		if id == "XMP " && c.xmp == nil && n > 0 && n <= scanLimit {
			buf := make([]byte, n)
			if _, err := io.ReadFull(f, buf); err == nil {
				c.xmp = buf
			}
		}
		off += 8 + n + (n & 1) // chunks are even aligned
	}
	return c
}

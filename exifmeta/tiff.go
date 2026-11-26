package exifmeta

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
)

const (
	tagImageDescription = 0x010E
	tagUserComment      = 0x9286

	tagExifIFD   = 0x8769
	tagGPSIFD    = 0x8825
	tagInterop   = 0xA005
	tagSubIFDs   = 0x014A
	typeASCII    = 2
	typeUndefine = 7
)

// tagNames labels the textual tags worth showing to a reader.
var tagNames = map[uint16]string{
	tagImageDescription: "ImageDescription",
	tagUserComment:      "UserComment",
	0x010F:              "Make",
	0x0110:              "Model",
	0x0112:              "Orientation",
	0x0131:              "Software",
	0x013B:              "Artist",
	0x8298:              "Copyright",
	0x9003:              "DateTimeOriginal",
	0x9C9B:              "XPTitle",
	0x9C9C:              "XPComment",
	0x9C9D:              "XPAuthor",
	0x9C9E:              "XPKeywords",
	0x9C9F:              "XPSubject",
}

func tagName(tag uint16) string {
	if name, ok := tagNames[tag]; ok {
		return name
	}
	return fmt.Sprintf("Tag 0x%04X", tag)
}

// sortedTags keeps the reported order stable across map iterations.
func sortedTags(tags map[uint16]string) []uint16 {
	out := make([]uint16, 0, len(tags))
	for tag := range tags {
		out = append(out, tag)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// tiffStrings walks IFD0, ExifIFD, GPS, Interop and SubIFDs and returns the
// decoded string values keyed by tag. Later duplicates do not overwrite the
// first hit, matching how readers usually surface these tags.
func tiffStrings(raw []byte) map[uint16]string {
	out := make(map[uint16]string, 8)
	if len(raw) < 8 {
		return out
	}

	var bo binary.ByteOrder
	switch string(raw[0:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return out
	}
	u16 := func(off int) (uint16, bool) {
		if off < 0 || off+2 > len(raw) {
			return 0, false
		}
		return bo.Uint16(raw[off:]), true
	}
	u32 := func(off int) (uint32, bool) {
		if off < 0 || off+4 > len(raw) {
			return 0, false
		}
		return bo.Uint32(raw[off:]), true
	}
	magic, ok := u16(2)
	if !ok || magic != 42 {
		return out
	}
	first, ok := u32(4)
	if !ok {
		return out
	}

	queue := []uint32{first}
	seen := make(map[uint32]bool, 8)
	for len(queue) > 0 && len(seen) < 32 {
		off := int(queue[0])
		queue = queue[1:]
		if off <= 0 || seen[uint32(off)] {
			continue
		}
		seen[uint32(off)] = true

		count, ok := u16(off)
		if !ok || count == 0 || count > 512 {
			continue
		}
		for i := 0; i < int(count); i++ {
			e := off + 2 + i*12
			tag, _ := u16(e)
			typ, ok := u16(e + 2)
			if !ok {
				break
			}
			cnt, ok := u32(e + 4)
			if !ok {
				break
			}
			val, ok := u32(e + 8)
			if !ok {
				break
			}

			if cnt > 0 && cnt <= 4096 {
				var data []byte
				switch {
				case cnt <= 4:
					if e+8+int(cnt) <= len(raw) {
						data = raw[e+8 : e+8+int(cnt)]
					}
				case int(val)+int(cnt) <= len(raw):
					data = raw[int(val) : int(val)+int(cnt)]
				}
				if data != nil {
					switch {
					case typ == typeASCII:
						if _, exists := out[tag]; !exists {
							out[tag] = string(data)
						}
					case typ == typeUndefine && tag == tagUserComment:
						if _, exists := out[tag]; !exists {
							out[tag] = decodeUserComment(data)
						}
					}
				}
			}

			switch tag {
			case tagExifIFD, tagGPSIFD, tagInterop:
				queue = append(queue, val)
			case tagSubIFDs:
				if typ == 4 && cnt == 1 {
					queue = append(queue, val)
				}
			}
		}
		if next, ok := u32(off + 2 + int(count)*12); ok && next != 0 {
			queue = append(queue, next)
		}
	}
	return out
}

// decodeUserComment strips the 8 byte character code prefix that the Exif
// spec mandates for UserComment and decodes the remainder.
func decodeUserComment(b []byte) string {
	if len(b) < 8 {
		return strings.TrimRight(string(b), "\x00")
	}
	prefix, body := string(b[:8]), b[8:]
	switch {
	case strings.HasPrefix(prefix, "ASCII"):
		return strings.TrimRight(string(body), "\x00")
	case strings.HasPrefix(prefix, "UNICODE"):
		var order binary.ByteOrder = binary.BigEndian
		if len(body) >= 2 {
			if body[0] == 0xFF && body[1] == 0xFE {
				order = binary.LittleEndian
				body = body[2:]
			} else if body[0] == 0xFE && body[1] == 0xFF {
				body = body[2:]
			}
		}
		units := make([]uint16, 0, len(body)/2)
		for i := 0; i+1 < len(body); i += 2 {
			units = append(units, order.Uint16(body[i:]))
		}
		return strings.TrimRight(string(utf16.Decode(units)), "\x00")
	case strings.HasPrefix(prefix, "JIS"):
		return ""
	}
	// Undefined prefix: most writers leave eight zero bytes and plain ASCII.
	return strings.TrimRight(string(body), "\x00")
}

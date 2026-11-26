package exifmeta

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
)

// xmpMaxTexts bounds how many text runs are inspected in one packet.
const xmpMaxTexts = 64

// applyXMP reports the packet verbatim and, when no other source already
// identified the generator, looks inside it for a known payload. XMP never
// overrides what PNG textual chunks or EXIF already provided.
func (m *Meta) applyXMP(raw []byte) {
	if len(raw) == 0 {
		return
	}

	if packet := strings.TrimSpace(strings.TrimRight(string(raw), "\x00")); packet != "" {
		m.Fields = append(m.Fields, Field{Source: "xmp", Key: "XMP", Value: packet})
	}
	if m.Kind != "" {
		return
	}

	for _, text := range xmpTexts(raw) {
		if isJSONObject(text) {
			switch comfyGraphKind(text) {
			case "prompt":
				if m.Prompt == nil {
					m.Prompt = json.RawMessage(text)
					m.Kind = KindComfyUI
				}
			case "workflow":
				if m.Workflow == nil {
					m.Workflow = json.RawMessage(text)
					m.Kind = KindComfyUI
				}
			}
			continue
		}
		if m.Parameters == "" && looksLikeParameters(text) {
			m.Parameters = text
			m.Kind = KindA1111
		}
	}
}

// comfyGraphKind tells apart the API prompt graph (nodes keyed by id, each
// with class_type) from the UI workflow (nodes/links arrays).
func comfyGraphKind(text string) string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return ""
	}
	if _, ok := obj["nodes"]; ok {
		return "workflow"
	}
	for _, value := range obj {
		var node struct {
			ClassType string `json:"class_type"`
		}
		if err := json.Unmarshal(value, &node); err == nil && node.ClassType != "" {
			return "prompt"
		}
	}
	return ""
}

// xmpTexts returns the character data and attribute values of a packet.
// Malformed XML yields whatever was decoded so far, since XMP written by
// generators is frequently not well formed.
func xmpTexts(raw []byte) []string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity

	var out []string
	add := func(s string) bool {
		if s = strings.TrimSpace(s); len(s) >= 8 {
			out = append(out, s)
		}
		return len(out) < xmpMaxTexts
	}
	for len(out) < xmpMaxTexts {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.CharData:
			if !add(string(t)) {
				return out
			}
		case xml.StartElement:
			for _, attr := range t.Attr {
				if !add(attr.Value) {
					return out
				}
			}
		}
	}
	return out
}

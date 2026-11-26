// Package exifmeta extracts generation metadata embedded in image files:
// ComfyUI workflow/prompt JSON, A1111 style "parameters" text and NovelAI
// alpha channel LSB payloads.
//
// Containers are parsed by hand (JPEG APP1, PNG textual chunks, WebP RIFF
// chunks) so the package needs nothing beyond the standard library and never
// depends on file extensions.
package exifmeta

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
)

// Kinds of generation metadata this package understands.
const (
	KindComfyUI = "comfyui"
	KindA1111   = "a1111"
	KindNovelAI = "nai"
)

// Source is the minimal file interface the parser needs. Both *os.File and
// afero.File satisfy it.
type Source interface {
	io.ReadSeeker
	Stat() (os.FileInfo, error)
}

// Meta describes what was found in an image.
type Meta struct {
	Format string `json:"format"`
	Kind   string `json:"kind"`
	// HasText reports whether any textual metadata was decoded. Every field is
	// surfaced regardless of whether its content was recognised.
	HasText bool `json:"hasText"`
	// HasWorkflow reports whether a known generator format was recognised.
	HasWorkflow bool            `json:"hasWorkflow"`
	Fields      []Field         `json:"fields,omitempty"`
	Workflow    json.RawMessage `json:"workflow,omitempty"`
	Prompt      json.RawMessage `json:"prompt,omitempty"`
	Parameters  string          `json:"parameters,omitempty"`
	Summary     *Summary        `json:"summary,omitempty"`
}

// Field is one textual value found in the file, verbatim.
type Field struct {
	Source string `json:"source"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

// Summary holds the fields worth showing before the raw JSON.
type Summary struct {
	Model     string  `json:"model,omitempty"`
	Sampler   string  `json:"sampler,omitempty"`
	Scheduler string  `json:"scheduler,omitempty"`
	Steps     int     `json:"steps,omitempty"`
	CFG       float64 `json:"cfg,omitempty"`
	// Seed stays a string: 64 bit seeds do not survive a JSON number round
	// trip through most clients.
	Seed     string `json:"seed,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Positive string `json:"positive,omitempty"`
	Negative string `json:"negative,omitempty"`
}

// Read scans f and returns the metadata found. A file without any recognised
// metadata yields a Meta with an empty Kind, not an error.
func Read(f Source) (*Meta, error) {
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}

	head := make([]byte, 12)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return nil, err
	}
	head = head[:n]

	format := sniff(head)
	meta := &Meta{Format: format.String()}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var c chunks
	switch format {
	case fmtJPEG:
		c = jpegChunks(f, st.Size())
	case fmtPNG:
		c = pngChunks(f, st.Size())
	case fmtWEBP:
		c = webpChunks(f, st.Size())
	default:
		return meta, nil
	}

	meta.apply(&c)
	if meta.Kind != "" {
		return meta, nil
	}
	// Textual metadata is absent; NovelAI hides its payload in the alpha
	// channel, which needs a full decode. Only pay that cost for PNG files
	// that are small enough to be worth it.
	if format == fmtPNG && st.Size() <= stealthSizeLimit {
		if payload := stealthPayload(f); payload != nil {
			meta.applyNovelAI(payload)
		}
	}
	return meta, nil
}

// textChunk is a PNG tEXt/iTXt entry in file order.
type textChunk struct {
	source string
	key    string
	value  string
}

// chunks collects everything extracted from a container before interpretation.
type chunks struct {
	tiff  []byte
	xmp   []byte
	texts []textChunk
}

func (m *Meta) apply(c *chunks) {
	for _, tc := range c.texts {
		value := strings.TrimRight(tc.value, "\x00")
		if strings.TrimSpace(value) == "" {
			continue
		}
		m.Fields = append(m.Fields, Field{Source: tc.source, Key: tc.key, Value: value})
		m.recognize(tc.key, value)
	}

	if len(c.tiff) > 0 {
		tags := tiffStrings(c.tiff)
		for _, tag := range sortedTags(tags) {
			value := strings.TrimRight(strings.TrimSpace(tags[tag]), "\x00")
			if value == "" {
				continue
			}
			m.Fields = append(m.Fields, Field{
				Source: "exif",
				Key:    tagName(tag),
				Value:  value,
			})
		}
		for _, tag := range []uint16{tagUserComment, tagImageDescription} {
			value := strings.TrimRight(strings.TrimSpace(tags[tag]), "\x00")
			if m.Parameters == "" && looksLikeParameters(value) {
				m.Parameters = value
				if m.Kind == "" {
					m.Kind = KindA1111
				}
			}
		}
	}

	m.applyXMP(c.xmp)

	switch {
	case m.Kind == KindComfyUI && m.Prompt != nil:
		m.Summary = comfySummary(m.Prompt)
	case m.Parameters != "":
		m.Summary = a1111Summary(m.Parameters)
	}

	m.HasWorkflow = m.Kind != ""
	m.HasText = len(m.Fields) > 0
}

// recognize labels the generator format behind a key/value pair. Recognition
// never influences whether the value is reported.
func (m *Meta) recognize(key, value string) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "workflow":
		if m.Workflow == nil && isJSONObject(value) {
			m.Workflow = json.RawMessage(value)
			m.Kind = KindComfyUI
		}
	case "prompt":
		if isJSONObject(value) {
			if m.Prompt == nil {
				m.Prompt = json.RawMessage(value)
				m.Kind = KindComfyUI
			}
			return
		}
		if m.Parameters == "" && looksLikeParameters(value) {
			m.Parameters = value
			if m.Kind == "" {
				m.Kind = KindA1111
			}
		}
	default:
		if m.Parameters == "" && looksLikeParameters(value) {
			m.Parameters = value
			if m.Kind == "" {
				m.Kind = KindA1111
			}
		}
	}
}

func (m *Meta) applyNovelAI(payload []byte) {
	m.Fields = append(m.Fields, Field{
		Source: "alpha-lsb",
		Key:    "payload",
		Value:  string(payload),
	})
	m.HasText = true
	if !isJSONObject(string(payload)) {
		return
	}
	m.Workflow = json.RawMessage(payload)
	m.Kind = KindNovelAI
	m.Summary = novelAISummary(payload)
	m.HasWorkflow = true
}

func isJSONObject(s string) bool {
	s = strings.TrimSpace(s)
	return strings.HasPrefix(s, "{") && json.Valid([]byte(s))
}

// looksLikeParameters reports whether text looks like A1111 style metadata
// rather than a plain caption.
func looksLikeParameters(s string) bool {
	ls := strings.ToLower(s)
	return strings.Contains(ls, "negative prompt") ||
		strings.Contains(ls, "steps:") ||
		strings.Contains(ls, "sampler:")
}

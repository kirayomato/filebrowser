package exifmeta

import (
	"encoding/binary"

	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pngWithText builds a minimal PNG carrying the given textual chunks, enough
// for pngChunks to walk; CRCs are not validated by the parser.
func pngWithText(t *testing.T, entries [][2]string) string {
	t.Helper()

	var b []byte
	b = append(b, "\x89PNG\r\n\x1a\n"...)

	chunk := func(typ string, data []byte) {
		var hdr [8]byte
		binary.BigEndian.PutUint32(hdr[:4], uint32(len(data)))
		copy(hdr[4:], typ)
		b = append(b, hdr[:]...)
		b = append(b, data...)
		b = append(b, 0, 0, 0, 0)
	}

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], 1)
	binary.BigEndian.PutUint32(ihdr[4:], 1)
	ihdr[8], ihdr[9] = 8, 6 // 8 bit RGBA
	chunk("IHDR", ihdr)
	for _, e := range entries {
		chunk("tEXt", append([]byte(e[0]+"\x00"), []byte(e[1])...))
	}
	chunk("IEND", nil)

	path := filepath.Join(t.TempDir(), "img.png")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) *Meta {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	meta, err := Read(f)
	if err != nil {
		t.Fatal(err)
	}
	return meta
}

func TestReadComfyUI(t *testing.T) {
	prompt := `{"3":{"class_type":"KSampler","inputs":{"seed":42,"steps":20,"cfg":7,"sampler_name":"euler","scheduler":"normal","positive":["6",0],"negative":["7",0]}},` +
		`"6":{"class_type":"CLIPTextEncode","inputs":{"text":"a cat"}},` +
		`"7":{"class_type":"CLIPTextEncode","inputs":{"text":"blurry"}},` +
		`"4":{"class_type":"CheckpointLoaderSimple","inputs":{"ckpt_name":"sd.safetensors"}},` +
		`"5":{"class_type":"EmptyLatentImage","inputs":{"width":512,"height":768}}}`
	workflow := `{"nodes":[],"links":[]}`

	meta := readFile(t, pngWithText(t, [][2]string{{"prompt", prompt}, {"workflow", workflow}}))
	if meta.Kind != KindComfyUI || !meta.HasWorkflow {
		t.Fatalf("kind = %q, hasWorkflow = %v", meta.Kind, meta.HasWorkflow)
	}
	if meta.Workflow == nil || meta.Prompt == nil {
		t.Fatalf("workflow/prompt not captured")
	}
	s := meta.Summary
	if s == nil {
		t.Fatal("summary is nil")
	}
	if s.Positive != "a cat" || s.Negative != "blurry" {
		t.Errorf("prompts = %q / %q", s.Positive, s.Negative)
	}
	if s.Model != "sd.safetensors" || s.Sampler != "euler" || s.Steps != 20 || s.CFG != 7 || s.Seed != "42" {
		t.Errorf("unexpected summary: %+v", s)
	}
	if s.Width != 512 || s.Height != 768 {
		t.Errorf("size = %dx%d", s.Width, s.Height)
	}
}

func TestReadA1111(t *testing.T) {
	params := "a cat\nNegative prompt: blurry\nSteps: 28, Sampler: DPM++ 2M, Schedule type: Karras, CFG scale: 5.5, Seed: 1234, Size: 832x1216, Model: animagine"
	meta := readFile(t, pngWithText(t, [][2]string{{"parameters", params}}))
	if meta.Kind != KindA1111 {
		t.Fatalf("kind = %q", meta.Kind)
	}
	s := meta.Summary
	if s == nil || s.Positive != "a cat" || s.Negative != "blurry" {
		t.Fatalf("summary = %+v", s)
	}
	if s.Steps != 28 || s.CFG != 5.5 || s.Seed != "1234" || s.Width != 832 || s.Height != 1216 {
		t.Errorf("unexpected summary: %+v", s)
	}
	if s.Sampler != "DPM++ 2M" || s.Scheduler != "Karras" || s.Model != "animagine" {
		t.Errorf("unexpected summary: %+v", s)
	}
}

func TestReadPlainImage(t *testing.T) {
	meta := readFile(t, pngWithText(t, [][2]string{{"Comment", "just a caption"}}))
	if meta.HasWorkflow || meta.Kind != "" || meta.Workflow != nil {
		t.Fatalf("plain image reported a generator format: %+v", meta)
	}
	// Unrecognised text is still reported verbatim.
	if !meta.HasText || len(meta.Fields) != 1 {
		t.Fatalf("fields = %+v", meta.Fields)
	}
	if f := meta.Fields[0]; f.Key != "Comment" || f.Value != "just a caption" || f.Source != "png-tEXt" {
		t.Errorf("unexpected field: %+v", f)
	}
}

// jpegWithXMP builds a minimal JPEG whose APP1 segment carries an XMP packet.
func jpegWithXMP(t *testing.T, xmp string) string {
	t.Helper()

	payload := append([]byte(xmpJPEGIdent), xmp...)
	var b []byte
	b = append(b, 0xFF, 0xD8)
	b = append(b, 0xFF, 0xE1, byte((len(payload)+2)>>8), byte(len(payload)+2))
	b = append(b, payload...)
	b = append(b, 0xFF, 0xD9)

	path := filepath.Join(t.TempDir(), "img.jpg")
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadXMPParameters(t *testing.T) {
	packet := `<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:description>` +
		`<rdf:Alt><rdf:li xml:lang="x-default">a cat&#10;Negative prompt: blurry&#10;Steps: 30, Sampler: Euler a, CFG scale: 6, Seed: 99</rdf:li></rdf:Alt>` +
		`</dc:description></rdf:Description></rdf:RDF></x:xmpmeta>`

	meta := readFile(t, jpegWithXMP(t, packet))
	if !meta.HasText || len(meta.Fields) != 1 || meta.Fields[0].Source != "xmp" {
		t.Fatalf("xmp field missing: %+v", meta.Fields)
	}
	if meta.Kind != KindA1111 || meta.Summary == nil {
		t.Fatalf("kind = %q summary = %+v", meta.Kind, meta.Summary)
	}
	if meta.Summary.Steps != 30 || meta.Summary.Seed != "99" || meta.Summary.Sampler != "Euler a" {
		t.Errorf("unexpected summary: %+v", meta.Summary)
	}
}

func TestReadXMPWorkflowJSON(t *testing.T) {
	// Attribute form, as some JPEG writers store the graph.
	graph := `{"3":{"class_type":"KSampler","inputs":{"steps":12}}}`
	packet := `<x:xmpmeta xmlns:x="adobe:ns:meta/"><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">` +
		`<rdf:Description xmlns:fb="https://filebrowser.org/ns/" fb:prompt="` + strings.ReplaceAll(graph, `"`, "&quot;") + `"/>` +
		`</rdf:RDF></x:xmpmeta>`

	meta := readFile(t, jpegWithXMP(t, packet))
	if meta.Kind != KindComfyUI || meta.Prompt == nil || meta.Summary == nil || meta.Summary.Steps != 12 {
		t.Fatalf("kind = %q summary = %+v", meta.Kind, meta.Summary)
	}
}

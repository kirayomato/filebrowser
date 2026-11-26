package exifmeta

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

// comfyNode is one entry of a ComfyUI API-format prompt graph.
type comfyNode struct {
	ClassType string         `json:"class_type"`
	Inputs    map[string]any `json:"inputs"`
}

// comfySummary pulls the common sampler fields out of a ComfyUI prompt graph.
// The UI workflow JSON is a different shape and is left to the client.
func comfySummary(raw []byte) *Summary {
	var nodes map[string]comfyNode
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // keep seeds exact
	if err := dec.Decode(&nodes); err != nil || len(nodes) == 0 {
		return nil
	}

	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	s := &Summary{}
	var sampler *comfyNode
	for _, id := range ids {
		if n := nodes[id]; strings.Contains(n.ClassType, "KSampler") {
			n := n
			sampler = &n
			break
		}
	}
	if sampler != nil {
		s.Steps = inputInt(sampler.Inputs, "steps")
		s.CFG = inputFloat(sampler.Inputs, "cfg")
		s.Sampler = inputString(sampler.Inputs, "sampler_name")
		s.Scheduler = inputString(sampler.Inputs, "scheduler")
		s.Seed = inputNumberString(sampler.Inputs, "seed")
		if s.Seed == "" {
			s.Seed = inputNumberString(sampler.Inputs, "noise_seed")
		}
		s.Positive = comfyText(nodes, sampler.Inputs["positive"])
		s.Negative = comfyText(nodes, sampler.Inputs["negative"])
	}

	for _, id := range ids {
		n := nodes[id]
		switch {
		case s.Model == "" && strings.Contains(n.ClassType, "CheckpointLoader"):
			s.Model = firstInputString(n.Inputs, "ckpt_name", "config_name")
		case s.Model == "" && strings.Contains(n.ClassType, "UNETLoader"):
			s.Model = firstInputString(n.Inputs, "unet_name")
		}
		if s.Width == 0 && strings.Contains(n.ClassType, "EmptyLatent") {
			s.Width = inputInt(n.Inputs, "width")
			s.Height = inputInt(n.Inputs, "height")
		}
	}

	if s.Positive == "" && s.Negative == "" {
		// Sampler nodes that do not reference CLIPTextEncode directly: fall
		// back to the text encoders in id order.
		var texts []string
		for _, id := range ids {
			n := nodes[id]
			if n.ClassType == "CLIPTextEncode" {
				if t := inputString(n.Inputs, "text"); t != "" {
					texts = append(texts, t)
				}
			}
			if len(texts) == 2 {
				break
			}
		}
		if len(texts) > 0 {
			s.Positive = texts[0]
		}
		if len(texts) > 1 {
			s.Negative = texts[1]
		}
	}

	if *s == (Summary{}) {
		return nil
	}
	return s
}

// comfyText resolves a link such as ["6", 0] to the text of that node.
func comfyText(nodes map[string]comfyNode, link any) string {
	ref, ok := link.([]any)
	if !ok || len(ref) == 0 {
		return ""
	}
	id, ok := ref[0].(string)
	if !ok {
		return ""
	}
	return inputString(nodes[id].Inputs, "text")
}

// a1111Summary parses the "parameters" text block written by A1111 and most
// compatible frontends.
func a1111Summary(text string) *Summary {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")

	paramsIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(lines[i]), "steps:") {
			paramsIdx = i
			break
		}
	}
	body, paramLine := lines, ""
	if paramsIdx >= 0 {
		body = lines[:paramsIdx]
		paramLine = strings.Join(lines[paramsIdx:], " ")
	}

	var pos, neg []string
	inNegative := false
	for _, line := range body {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "negative prompt:") {
			inNegative = true
			if _, rest, found := strings.Cut(line, ":"); found {
				if rest = strings.TrimSpace(rest); rest != "" {
					neg = append(neg, rest)
				}
			}
			continue
		}
		if inNegative {
			neg = append(neg, line)
		} else {
			pos = append(pos, line)
		}
	}

	s := &Summary{
		Positive: strings.TrimSpace(strings.Join(pos, "\n")),
		Negative: strings.TrimSpace(strings.Join(neg, "\n")),
	}
	for _, pair := range splitParams(paramLine) {
		key, value, found := strings.Cut(pair, ":")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.Trim(strings.TrimSpace(value), `"`)
		switch key {
		case "steps":
			s.Steps, _ = strconv.Atoi(value)
		case "sampler":
			s.Sampler = value
		case "schedule type", "scheduler":
			if s.Scheduler == "" {
				s.Scheduler = value
			}
		case "cfg scale":
			s.CFG, _ = strconv.ParseFloat(value, 64)
		case "seed":
			s.Seed = value
		case "size":
			w, h, _ := strings.Cut(value, "x")
			s.Width, _ = strconv.Atoi(strings.TrimSpace(w))
			s.Height, _ = strconv.Atoi(strings.TrimSpace(h))
		case "model":
			if s.Model == "" {
				s.Model = value
			}
		}
	}
	return s
}

// splitParams splits "a: 1, b: 2" while keeping commas inside quotes.
func splitParams(s string) []string {
	var out []string
	var b strings.Builder
	inQuote := false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			b.WriteRune(r)
		case r == ',' && !inQuote:
			out = append(out, b.String())
			b.Reset()
		default:
			b.WriteRune(r)
		}
	}
	if b.Len() > 0 {
		out = append(out, b.String())
	}
	return out
}

// novelAIPayload is the JSON NovelAI hides in the alpha channel.
type novelAIPayload struct {
	Description string `json:"Description"`
	Source      string `json:"Source"`
	Comment     string `json:"Comment"`
}

type novelAIComment struct {
	Prompt        string  `json:"prompt"`
	UC            string  `json:"uc"`
	Steps         int     `json:"steps"`
	Scale         float64 `json:"scale"`
	Seed          int64   `json:"seed"`
	Sampler       string  `json:"sampler"`
	NoiseSchedule string  `json:"noise_schedule"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
}

func novelAISummary(raw []byte) *Summary {
	var payload novelAIPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}

	s := &Summary{Positive: strings.TrimSpace(payload.Description)}
	if payload.Comment != "" {
		var c novelAIComment
		if err := json.Unmarshal([]byte(payload.Comment), &c); err == nil {
			if s.Positive == "" {
				s.Positive = strings.TrimSpace(c.Prompt)
			}
			s.Negative = strings.TrimSpace(c.UC)
			s.Steps = c.Steps
			s.CFG = c.Scale
			if c.Seed != 0 {
				s.Seed = strconv.FormatInt(c.Seed, 10)
			}
			s.Sampler = c.Sampler
			s.Scheduler = c.NoiseSchedule
			s.Width = c.Width
			s.Height = c.Height
		}
	}
	if *s == (Summary{}) {
		return nil
	}
	return s
}

func inputString(in map[string]any, key string) string {
	if v, ok := in[key].(string); ok {
		return v
	}
	return ""
}

func firstInputString(in map[string]any, keys ...string) string {
	for _, key := range keys {
		if v := inputString(in, key); v != "" {
			return v
		}
	}
	return ""
}

func inputInt(in map[string]any, key string) int {
	switch v := in[key].(type) {
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	}
	return 0
}

func inputFloat(in map[string]any, key string) float64 {
	switch v := in[key].(type) {
	case float64:
		return v
	case json.Number:
		f, _ := v.Float64()
		return f
	}
	return 0
}

// inputNumberString returns the numeric input verbatim so 64 bit seeds do not
// lose digits.
func inputNumberString(in map[string]any, key string) string {
	switch v := in[key].(type) {
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case string:
		return v
	}
	return ""
}

package onlineconf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode"
	"unicode/utf16"
)

// HomeBlocksError describes a field the admin can correct before publishing.
type HomeBlocksError string

func (e HomeBlocksError) Error() string { return "onlineconf: frontend.home_blocks: " + string(e) }

func validateHomeBlocks(raw json.RawMessage) error {
	var blocks []map[string]any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&blocks); err != nil || blocks == nil || len(blocks) > 6 {
		return HomeBlocksError("expected an array of up to 6 block objects")
	}
	for i, block := range blocks {
		if problem := homeBlockProblem(block); problem != "" {
			return HomeBlocksError(fmt.Sprintf("block %d: %s", i+1, problem))
		}
	}
	return nil
}

func homeBlockProblem(block map[string]any) string {
	switch block["type"] {
	case "collection":
		number, ok := block["id"].(json.Number)
		id, err := number.Float64()
		if !ok || err != nil || id < 1 || id > float64(MaxSafeID) || math.Trunc(id) != id {
			return "id must be a positive safe integer"
		}
	case "banner":
		if !validBlockText(block["title"], 100) {
			return "title must contain 1 to 100 characters"
		}
		if body, present := block["text"]; present && !validBlockText(body, 300) {
			return "text must contain 1 to 300 characters"
		}
		if value, present := block["action"]; present {
			action, _ := value.(map[string]any)
			if !validBlockText(action["label"], 40) || !validBlockLink(action["href"]) {
				return "action requires a label of up to 40 characters and an internal product path"
			}
		}
	default:
		return "type must be banner or collection"
	}
	return ""
}

// Match JavaScript String.trim() and UTF-16 string.length in the SPA.
func jsSpace(r rune) bool { return r == '\uFEFF' || (r != '\u0085' && unicode.IsSpace(r)) }

func validBlockText(value any, limit int) bool {
	text, ok := value.(string)
	trimmed := strings.TrimFunc(text, jsSpace)
	length := 0
	for _, r := range trimmed {
		length += utf16.RuneLen(r)
	}
	return ok && length > 0 && length <= limit
}

func validBlockLink(value any) bool {
	href, ok := value.(string)
	if !ok || !strings.HasPrefix(href, "/") || strings.HasPrefix(href, "//") {
		return false
	}
	for _, r := range href {
		if r == '\\' || jsSpace(r) || unicode.IsControl(r) {
			return false
		}
	}

	path, _, _ := strings.Cut(href[1:], "?")
	path, _, _ = strings.Cut(path, "#")
	parts := strings.Split(path, "/")
	segments := make([]string, 0, len(parts))
	for i, part := range parts {
		// Browsers resolve percent-encoded dots, but preserve empty segments.
		switch strings.ToLower(part) {
		case ".", "%2e":
		case "..", ".%2e", "%2e.", "%2e%2e":
			if len(segments) > 0 {
				segments = segments[:len(segments)-1]
			}
		default:
			segments = append(segments, part)
			continue
		}
		if i == len(parts)-1 {
			segments = append(segments, "")
		}
	}
	return segments[0] != "onlineconf" && !(segments[0] == "" && len(segments) > 1)
}

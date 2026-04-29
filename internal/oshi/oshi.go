package oshi

import "strings"

const (
	MetadataPublicKey = "oshi_color"
	DefaultColor      = "#b2b2ff"
)

var Palette = []string{
	"#ffb2b2",
	"#ffb2d8",
	"#ffb2ff",
	"#d8b2ff",
	"#b2b2ff",
	"#b2d8ff",
	"#b2ffff",
	"#b2ffd8",
	"#b2ffb2",
	"#d8ffb2",
	"#ffffb2",
	"#ffd8b2",
}

func NormalizeColor(raw string) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	for _, candidate := range Palette {
		if normalized == candidate {
			return candidate
		}
	}
	return ""
}


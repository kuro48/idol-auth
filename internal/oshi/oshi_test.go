package oshi

import "testing"

func TestNormalizeColor(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "exact", raw: "#b2d8ff", want: "#b2d8ff"},
		{name: "trim and lowercase", raw: "  #FFB2D8 ", want: "#ffb2d8"},
		{name: "invalid", raw: "#123456", want: ""},
		{name: "empty", raw: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeColor(tt.raw); got != tt.want {
				t.Fatalf("NormalizeColor(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

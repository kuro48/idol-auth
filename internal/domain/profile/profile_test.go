package profile_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ryunosukekurokawa/idol-auth/internal/domain/profile"
)

// ---- FanYears ----

func TestFanYears_FromYearOnly(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		fanSince string
		want     int
	}{
		{"2019", 7},
		{"2026", 0},
		{"2025", 1},
		{"2000", 26},
	}
	for _, tc := range cases {
		t.Run(tc.fanSince, func(t *testing.T) {
			got := profile.FanYears(tc.fanSince, now)
			if got != tc.want {
				t.Errorf("FanYears(%q) = %d, want %d", tc.fanSince, got, tc.want)
			}
		})
	}
}

func TestFanYears_FromYearMonth(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		fanSince string
		want     int
	}{
		{"2019-04", 7},  // 2019-04 → 7 full years by 2026-06
		{"2026-06", 0},  // same month
		{"2026-07", 0},  // future month → 0
		{"2025-06", 1},
		{"2020-12", 5},  // 2020-12 → 5 full years by 2026-06
	}
	for _, tc := range cases {
		t.Run(tc.fanSince, func(t *testing.T) {
			got := profile.FanYears(tc.fanSince, now)
			if got != tc.want {
				t.Errorf("FanYears(%q) = %d, want %d", tc.fanSince, got, tc.want)
			}
		})
	}
}

func TestFanYears_InvalidInputReturnsZero(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []string{"", "abcd", "2019-13", "2019-00", "19", "2019-4", "2019-04-01"}
	for _, s := range cases {
		t.Run(s+"_invalid", func(t *testing.T) {
			got := profile.FanYears(s, now)
			if got != 0 {
				t.Errorf("FanYears(%q) = %d, want 0 for invalid input", s, got)
			}
		})
	}
}

func TestFanYears_FutureYearReturnsZero(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got := profile.FanYears("2027", now)
	if got != 0 {
		t.Errorf("FanYears(future year) = %d, want 0", got)
	}
}

// ---- ValidateDisplayName ----

func TestValidateDisplayName_Valid(t *testing.T) {
	cases := []string{"推し活太郎", "A", strings.Repeat("あ", 50)}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if err := profile.ValidateDisplayName(name); err != nil {
				t.Errorf("ValidateDisplayName(%q) unexpected error: %v", name, err)
			}
		})
	}
}

func TestValidateDisplayName_Invalid(t *testing.T) {
	cases := []struct {
		name string
		desc string
	}{
		{"", "empty"},
		{"   ", "whitespace only"},
		{strings.Repeat("あ", 51), "too long (>50 runes)"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if err := profile.ValidateDisplayName(tc.name); err == nil {
				t.Errorf("ValidateDisplayName(%q) expected error, got nil", tc.name)
			}
		})
	}
}

// ---- ValidateOshiIDs ----

func TestValidateOshiIDs_Valid(t *testing.T) {
	cases := [][]string{
		nil,
		{},
		{"member-01"},
		{"member-01", "member-03"},
		{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, // 10 items (max)
	}
	for _, ids := range cases {
		if err := profile.ValidateOshiIDs(ids); err != nil {
			t.Errorf("ValidateOshiIDs(%v) unexpected error: %v", ids, err)
		}
	}
}

func TestValidateOshiIDs_Invalid(t *testing.T) {
	cases := []struct {
		ids  []string
		desc string
	}{
		{[]string{""}, "empty string element"},
		{[]string{"member-01", ""}, "empty string at end"},
		{[]string{"   "}, "whitespace-only element"},
		{[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}, "11 items (over max 10)"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if err := profile.ValidateOshiIDs(tc.ids); err == nil {
				t.Errorf("ValidateOshiIDs(%v) expected error, got nil", tc.ids)
			}
		})
	}
}

// ---- ValidateFanSince ----

func TestValidateFanSince_Valid(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	cases := []string{
		"",        // optional field
		"2019",
		"2019-04",
		"2026",    // current year
		"2026-06", // current month
		"1990",
		"1990-01",
	}
	for _, s := range cases {
		t.Run(s+"_valid", func(t *testing.T) {
			if err := profile.ValidateFanSince(s, now); err != nil {
				t.Errorf("ValidateFanSince(%q) unexpected error: %v", s, err)
			}
		})
	}
}

func TestValidateFanSince_Invalid(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		s    string
		desc string
	}{
		{"2027", "future year"},
		{"2026-07", "future month"},
		{"abcd", "non-numeric"},
		{"2019-13", "invalid month 13"},
		{"2019-00", "invalid month 00"},
		{"19", "2-digit year"},
		{"2019-4", "single-digit month not zero-padded"},
		{"2019-04-01", "full date not allowed"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			if err := profile.ValidateFanSince(tc.s, now); err == nil {
				t.Errorf("ValidateFanSince(%q) expected error, got nil", tc.s)
			}
		})
	}
}

// ---- Profile struct ----

func TestProfile_ComputeFanYears(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	p := profile.Profile{FanSince: "2020-01"}
	got := p.ComputeFanYears(now)
	if got != 6 {
		t.Errorf("ComputeFanYears = %d, want 6", got)
	}
}

func TestProfile_ComputeFanYears_EmptyFanSince(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	p := profile.Profile{FanSince: ""}
	got := p.ComputeFanYears(now)
	if got != 0 {
		t.Errorf("ComputeFanYears with empty FanSince = %d, want 0", got)
	}
}

func TestProfile_PublicView_ExcludesPII(t *testing.T) {
	p := profile.Profile{
		IdentityID:  "id-1",
		DisplayName: "推し活太郎",
		Email:       "user@example.com",
		Phone:       "+81-90-0000-0000",
		OshiColor:   "#ffb2d8",
		OshiIDs:     []string{"member-01"},
		FanSince:    "2019",
	}
	pub := p.PublicView()
	if pub.Email != "" || pub.Phone != "" {
		t.Errorf("PublicView must not expose PII: email=%q phone=%q", pub.Email, pub.Phone)
	}
	if pub.IdentityID != p.IdentityID {
		t.Errorf("PublicView.IdentityID = %q, want %q", pub.IdentityID, p.IdentityID)
	}
	if pub.DisplayName != p.DisplayName {
		t.Errorf("PublicView.DisplayName = %q, want %q", pub.DisplayName, p.DisplayName)
	}
	if pub.OshiColor != p.OshiColor {
		t.Errorf("PublicView.OshiColor = %q, want %q", pub.OshiColor, p.OshiColor)
	}
	if len(pub.OshiIDs) != len(p.OshiIDs) {
		t.Errorf("PublicView.OshiIDs length mismatch: got %d, want %d", len(pub.OshiIDs), len(p.OshiIDs))
	}
}

// ---- MetadataPublic encoding ----

func TestMetadataPublic_RoundTrip(t *testing.T) {
	orig := profile.MetadataPublic{
		OshiColor: "#ffb2d8",
		OshiIDs:   []string{"member-01", "member-03"},
		FanSince:  "2019-04",
	}
	data, err := orig.Marshal()
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}
	var decoded profile.MetadataPublic
	if err := decoded.Unmarshal(data); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}
	if decoded.OshiColor != orig.OshiColor {
		t.Errorf("OshiColor: got %q, want %q", decoded.OshiColor, orig.OshiColor)
	}
	if len(decoded.OshiIDs) != len(orig.OshiIDs) {
		t.Errorf("OshiIDs length: got %d, want %d", len(decoded.OshiIDs), len(orig.OshiIDs))
	}
	if decoded.FanSince != orig.FanSince {
		t.Errorf("FanSince: got %q, want %q", decoded.FanSince, orig.FanSince)
	}
}

func TestMetadataPublic_Unmarshal_EmptyJSON(t *testing.T) {
	var m profile.MetadataPublic
	if err := m.Unmarshal([]byte("{}")); err != nil {
		t.Fatalf("Unmarshal empty JSON error: %v", err)
	}
	if m.OshiColor != "" || len(m.OshiIDs) != 0 || m.FanSince != "" {
		t.Errorf("Expected zero-value MetadataPublic, got %+v", m)
	}
}

func TestMetadataPublic_Unmarshal_NilReturnsZero(t *testing.T) {
	var m profile.MetadataPublic
	if err := m.Unmarshal(nil); err != nil {
		t.Fatalf("Unmarshal nil error: %v", err)
	}
}

package profile_test

import (
	"strings"
	"testing"
	"time"

	"github.com/kuro48/idol-auth/internal/domain/profile"
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
		{"2019-04", 7}, // 2019-04 → 7 full years by 2026-06
		{"2026-06", 0}, // same month
		{"2026-07", 0}, // future month → 0
		{"2025-06", 1},
		{"2020-12", 5}, // 2020-12 → 5 full years by 2026-06
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
		"", // optional field
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

func TestValidateAvatarURL(t *testing.T) {
	for _, value := range []string{"", "https://example.com/avatar.png", "http://localhost:3000/avatar.png"} {
		if err := profile.ValidateAvatarURL(value); err != nil {
			t.Fatalf("ValidateAvatarURL(%q) unexpected error: %v", value, err)
		}
	}
	for _, value := range []string{"javascript:alert(1)", "ftp://example.com/a.png", "https://" + strings.Repeat("a", 2100)} {
		if err := profile.ValidateAvatarURL(value); err == nil {
			t.Fatalf("ValidateAvatarURL(%q) expected error", value)
		}
	}
}

func TestValidateLocaleAndTimezone(t *testing.T) {
	for _, value := range []string{"", "ja-JP", "en-US"} {
		if err := profile.ValidateLocale(value); err != nil {
			t.Fatalf("ValidateLocale(%q) unexpected error: %v", value, err)
		}
	}
	for _, value := range []string{"japanese", "ja_jp", strings.Repeat("a", 40)} {
		if err := profile.ValidateLocale(value); err == nil {
			t.Fatalf("ValidateLocale(%q) expected error", value)
		}
	}
	for _, value := range []string{"", "Asia/Tokyo", "UTC"} {
		if err := profile.ValidateTimezone(value); err != nil {
			t.Fatalf("ValidateTimezone(%q) unexpected error: %v", value, err)
		}
	}
	if err := profile.ValidateTimezone("Mars/Base"); err == nil {
		t.Fatal("ValidateTimezone(Mars/Base) expected error")
	}
}

func TestValidateBirthdate(t *testing.T) {
	now := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	for _, value := range []string{"", "2000-01-02", "2026-05-14"} {
		if err := profile.ValidateBirthdate(value, now); err != nil {
			t.Fatalf("ValidateBirthdate(%q) unexpected error: %v", value, err)
		}
	}
	for _, value := range []string{"2000-1-2", "2026-05-15", "1800-01-01"} {
		if err := profile.ValidateBirthdate(value, now); err == nil {
			t.Fatalf("ValidateBirthdate(%q) expected error", value)
		}
	}
}

func TestValidateBadgesAndContribution(t *testing.T) {
	badges := []profile.Badge{{
		ID:          "top_contributor_2026",
		Label:       "Top Contributor 2026",
		Description: "投稿型サービスへの高い貢献",
		SourceAppID: "app-1",
		Level:       "gold",
		IssuedAt:    "2026-05-14T00:00:00Z",
	}}
	if err := profile.ValidateBadges(badges); err != nil {
		t.Fatalf("ValidateBadges() unexpected error: %v", err)
	}
	if err := profile.ValidatePrimaryBadgeID("top_contributor_2026", badges); err != nil {
		t.Fatalf("ValidatePrimaryBadgeID() unexpected error: %v", err)
	}
	if err := profile.ValidatePrimaryBadgeID("missing", badges); err == nil {
		t.Fatal("ValidatePrimaryBadgeID(missing) expected error")
	}
	if err := profile.ValidateBadges([]profile.Badge{{ID: "", Label: "No ID"}}); err == nil {
		t.Fatal("ValidateBadges(empty id) expected error")
	}
	if err := profile.ValidateContributionScore(-1); err == nil {
		t.Fatal("ValidateContributionScore(-1) expected error")
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
		IdentityID:              "id-1",
		DisplayName:             "推し活太郎",
		Email:                   "user@example.com",
		Phone:                   "+81-90-0000-0000",
		OshiColor:               "#ffb2d8",
		OshiIDs:                 []string{"member-01"},
		FanSince:                "2019",
		Birthdate:               "2000-01-02",
		NotificationPreferences: profile.NotificationPreferences{EmailEnabled: true},
		Badges:                  []profile.Badge{{ID: "top", Label: "Top"}},
		PrimaryBadgeID:          "top",
		ContributionScore:       10,
	}
	pub := p.PublicView()
	if pub.Email != "" || pub.Phone != "" || pub.Birthdate != "" || pub.NotificationPreferences.EmailEnabled {
		t.Errorf("PublicView exposed private fields: %+v", pub)
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
	if len(pub.Badges) != 1 || pub.PrimaryBadgeID != "top" || pub.ContributionScore != 10 {
		t.Errorf("PublicView should expose public badge fields, got %+v", pub)
	}
}

// ---- MetadataPublic encoding ----

func TestMetadataPublic_RoundTrip(t *testing.T) {
	orig := profile.MetadataPublic{
		OshiColor:         "#ffb2d8",
		OshiIDs:           []string{"member-01", "member-03"},
		FanSince:          "2019-04",
		AvatarURL:         "https://example.com/avatar.png",
		Locale:            "ja-JP",
		Timezone:          "Asia/Tokyo",
		Badges:            []profile.Badge{{ID: "top", Label: "Top"}},
		PrimaryBadgeID:    "top",
		ContributionScore: 12,
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
	if decoded.AvatarURL != orig.AvatarURL || decoded.Locale != orig.Locale || decoded.Timezone != orig.Timezone {
		t.Errorf("profile metadata fields mismatch: %+v", decoded)
	}
	if len(decoded.Badges) != 1 || decoded.PrimaryBadgeID != "top" || decoded.ContributionScore != 12 {
		t.Errorf("badge metadata mismatch: %+v", decoded)
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

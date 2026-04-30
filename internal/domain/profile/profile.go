package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxDisplayNameRunes = 50
	maxOshiIDs          = 10
)

// Profile holds the shared account profile returned by the profile API.
// Email and Phone are PII and must be stripped before sending to third-party apps.
type Profile struct {
	IdentityID  string   `json:"identity_id"`
	DisplayName string   `json:"display_name,omitempty"`
	OshiColor   string   `json:"oshi_color,omitempty"`
	OshiIDs     []string `json:"oshi_ids,omitempty"`
	FanSince    string   `json:"fan_since,omitempty"`
	// PII — excluded from PublicView
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

// PublicView returns a copy of the profile without PII fields.
// Use this when responding to app management-token requests.
func (p Profile) PublicView() Profile {
	return Profile{
		IdentityID:  p.IdentityID,
		DisplayName: p.DisplayName,
		OshiColor:   p.OshiColor,
		OshiIDs:     p.OshiIDs,
		FanSince:    p.FanSince,
	}
}

// ComputeFanYears returns the number of full years since FanSince relative to now.
// Returns 0 if FanSince is empty or invalid, or if the result would be negative.
func (p Profile) ComputeFanYears(now time.Time) int {
	return FanYears(p.FanSince, now)
}

// FanYears parses fanSince ("YYYY" or "YYYY-MM") and returns full years elapsed
// relative to now. Returns 0 for invalid or future values.
func FanYears(fanSince string, now time.Time) int {
	start, ok := parseFanSince(fanSince)
	if !ok {
		return 0
	}
	years := now.Year() - start.Year()
	// Subtract one if the anniversary hasn't occurred yet this year.
	anniversary := start.AddDate(years, 0, 0)
	if now.Before(anniversary) {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// ValidateDisplayName returns an error if name is empty (after trimming) or exceeds 50 runes.
func ValidateDisplayName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("display_name must not be empty")
	}
	if utf8.RuneCountInString(name) > maxDisplayNameRunes {
		return fmt.Errorf("display_name must be at most %d characters", maxDisplayNameRunes)
	}
	return nil
}

// ValidateOshiIDs returns an error if any element is blank or the slice exceeds 10 entries.
func ValidateOshiIDs(ids []string) error {
	if len(ids) > maxOshiIDs {
		return fmt.Errorf("oshi_ids must contain at most %d entries", maxOshiIDs)
	}
	for i, id := range ids {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("oshi_ids[%d] must not be empty", i)
		}
	}
	return nil
}

// ValidateFanSince returns an error if s is not empty and does not conform to
// "YYYY" or "YYYY-MM" format, or if the value is in the future relative to now.
func ValidateFanSince(s string, now time.Time) error {
	if s == "" {
		return nil
	}
	start, ok := parseFanSince(s)
	if !ok {
		return fmt.Errorf("fan_since must be in YYYY or YYYY-MM format, got %q", s)
	}
	// Allow the current month: compare start against first day of current month.
	nowMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	if start.After(nowMonth) {
		return fmt.Errorf("fan_since must not be in the future (got %q)", s)
	}
	return nil
}

// parseFanSince parses "YYYY" or "YYYY-MM" and returns the first day of that
// period as UTC time. Returns false if the format is invalid.
func parseFanSince(s string) (time.Time, bool) {
	switch len(s) {
	case 4:
		y, err := strconv.Atoi(s)
		if err != nil {
			return time.Time{}, false
		}
		return time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC), true
	case 7:
		if s[4] != '-' {
			return time.Time{}, false
		}
		y, err := strconv.Atoi(s[:4])
		if err != nil {
			return time.Time{}, false
		}
		m, err := strconv.Atoi(s[5:])
		if err != nil || m < 1 || m > 12 {
			return time.Time{}, false
		}
		return time.Date(y, time.Month(m), 1, 0, 0, 0, 0, time.UTC), true
	default:
		return time.Time{}, false
	}
}

// UpdateInput carries the subset of profile fields the caller wants to change.
// A nil pointer means "leave unchanged"; a non-nil pointer is written.
type UpdateInput struct {
	DisplayName *string
	OshiColor   *string
	OshiIDs     *[]string
	FanSince    *string
}

// MetadataPublic is the structured representation of Kratos identity metadata_public.
// It extends the existing oshi_color key with oshi_ids and fan_since.
type MetadataPublic struct {
	OshiColor string   `json:"oshi_color,omitempty"`
	OshiIDs   []string `json:"oshi_ids,omitempty"`
	FanSince  string   `json:"fan_since,omitempty"`
}

// Marshal serialises MetadataPublic to JSON bytes.
func (m MetadataPublic) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

// Unmarshal deserialises JSON bytes into MetadataPublic.
// A nil or empty slice is treated as an empty object.
func (m *MetadataPublic) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, m)
}

package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxDisplayNameRunes = 50
	maxOshiIDs          = 10
	maxAvatarURLLength  = 2048
	maxBadgeCount       = 100
	maxBadgeIDLength    = 80
	maxBadgeLabelRunes  = 80
)

var localePattern = regexp.MustCompile(`^[a-z]{2,3}(-[A-Z]{2})?$`)

// Profile holds the shared account profile returned by the profile API.
// Email and Phone are PII and must be stripped before sending to third-party apps.
type Profile struct {
	IdentityID              string                  `json:"identity_id"`
	DisplayName             string                  `json:"display_name,omitempty"`
	AvatarURL               string                  `json:"avatar_url,omitempty"`
	Locale                  string                  `json:"locale,omitempty"`
	Timezone                string                  `json:"timezone,omitempty"`
	OshiColor               string                  `json:"oshi_color,omitempty"`
	OshiIDs                 []string                `json:"oshi_ids,omitempty"`
	FanSince                string                  `json:"fan_since,omitempty"`
	Badges                  []Badge                 `json:"badges,omitempty"`
	PrimaryBadgeID          string                  `json:"primary_badge_id,omitempty"`
	ContributionScore       int                     `json:"contribution_score,omitempty"`
	ContributionSummary     ContributionSummary     `json:"contribution_summary,omitempty"`
	Birthdate               string                  `json:"birthdate,omitempty"`
	NotificationPreferences NotificationPreferences `json:"notification_preferences,omitempty"`
	// PII — excluded from PublicView
	Email string `json:"email,omitempty"`
	Phone string `json:"phone,omitempty"`
}

type NotificationPreferences struct {
	EmailEnabled               bool `json:"email_enabled"`
	ProductUpdates             bool `json:"product_updates"`
	SecurityAlerts             bool `json:"security_alerts"`
	CommunityNotifications     bool `json:"community_notifications"`
	AppMembershipNotifications bool `json:"app_membership_notifications"`
}

type Badge struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	SourceAppID string `json:"source_app_id,omitempty"`
	Level       string `json:"level,omitempty"`
	IssuedAt    string `json:"issued_at,omitempty"`
}

type ContributionSummary struct {
	PostsCount                 int    `json:"posts_count,omitempty"`
	AcceptedContributionsCount int    `json:"accepted_contributions_count,omitempty"`
	HelpfulVotesCount          int    `json:"helpful_votes_count,omitempty"`
	LastContributedAt          string `json:"last_contributed_at,omitempty"`
}

// PublicView returns a copy of the profile without PII fields.
// Use this when responding to app management-token requests.
func (p Profile) PublicView() Profile {
	return Profile{
		IdentityID:          p.IdentityID,
		DisplayName:         p.DisplayName,
		AvatarURL:           p.AvatarURL,
		Locale:              p.Locale,
		Timezone:            p.Timezone,
		OshiColor:           p.OshiColor,
		OshiIDs:             p.OshiIDs,
		FanSince:            p.FanSince,
		Badges:              p.Badges,
		PrimaryBadgeID:      p.PrimaryBadgeID,
		ContributionScore:   p.ContributionScore,
		ContributionSummary: p.ContributionSummary,
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

func ValidateAvatarURL(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if len(s) > maxAvatarURLLength {
		return fmt.Errorf("avatar_url must be at most %d bytes", maxAvatarURLLength)
	}
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("avatar_url must be an absolute http or https URL")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("avatar_url must use http or https")
	}
	return nil
}

func ValidateLocale(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if !localePattern.MatchString(s) {
		return errors.New("locale must be a BCP 47 language tag like ja-JP")
	}
	return nil
}

func ValidateTimezone(s string) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	if _, err := time.LoadLocation(s); err != nil {
		return errors.New("timezone must be a valid IANA timezone")
	}
	return nil
}

func ValidateBirthdate(s string, now time.Time) error {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	d, err := time.Parse("2006-01-02", s)
	if err != nil {
		return errors.New("birthdate must be in YYYY-MM-DD format")
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if d.After(today) {
		return errors.New("birthdate must not be in the future")
	}
	if d.Year() < 1900 {
		return errors.New("birthdate is too old")
	}
	return nil
}

func ValidateBadges(badges []Badge) error {
	if len(badges) > maxBadgeCount {
		return fmt.Errorf("badges must contain at most %d entries", maxBadgeCount)
	}
	seen := make(map[string]struct{}, len(badges))
	for i, b := range badges {
		id := strings.TrimSpace(b.ID)
		if id == "" {
			return fmt.Errorf("badges[%d].id must not be empty", i)
		}
		if len(id) > maxBadgeIDLength {
			return fmt.Errorf("badges[%d].id must be at most %d bytes", i, maxBadgeIDLength)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("badges[%d].id is duplicated", i)
		}
		seen[id] = struct{}{}
		if strings.TrimSpace(b.Label) == "" {
			return fmt.Errorf("badges[%d].label must not be empty", i)
		}
		if utf8.RuneCountInString(b.Label) > maxBadgeLabelRunes {
			return fmt.Errorf("badges[%d].label must be at most %d characters", i, maxBadgeLabelRunes)
		}
		if b.IssuedAt != "" {
			if _, err := time.Parse(time.RFC3339, b.IssuedAt); err != nil {
				return fmt.Errorf("badges[%d].issued_at must be RFC3339", i)
			}
		}
	}
	return nil
}

func ValidatePrimaryBadgeID(id string, badges []Badge) error {
	if strings.TrimSpace(id) == "" {
		return nil
	}
	for _, b := range badges {
		if b.ID == id {
			return nil
		}
	}
	return errors.New("primary_badge_id must refer to an existing badge")
}

func ValidateContributionScore(score int) error {
	if score < 0 {
		return errors.New("contribution_score must not be negative")
	}
	return nil
}

func ValidateContributionSummary(summary ContributionSummary) error {
	if summary.PostsCount < 0 ||
		summary.AcceptedContributionsCount < 0 ||
		summary.HelpfulVotesCount < 0 {
		return errors.New("contribution_summary counts must not be negative")
	}
	if summary.LastContributedAt != "" {
		if _, err := time.Parse(time.RFC3339, summary.LastContributedAt); err != nil {
			return errors.New("contribution_summary.last_contributed_at must be RFC3339")
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
	DisplayName             *string
	AvatarURL               *string
	Locale                  *string
	Timezone                *string
	Birthdate               *string
	NotificationPreferences *NotificationPreferences
	OshiColor               *string
	OshiIDs                 *[]string
	FanSince                *string
	Badges                  *[]Badge
	PrimaryBadgeID          *string
	ContributionScore       *int
	ContributionSummary     *ContributionSummary
}

// MetadataPublic is the structured representation of Kratos identity metadata_public.
// It extends the existing oshi_color key with oshi_ids and fan_since.
type MetadataPublic struct {
	OshiColor           string              `json:"oshi_color,omitempty"`
	OshiIDs             []string            `json:"oshi_ids,omitempty"`
	FanSince            string              `json:"fan_since,omitempty"`
	AvatarURL           string              `json:"avatar_url,omitempty"`
	Locale              string              `json:"locale,omitempty"`
	Timezone            string              `json:"timezone,omitempty"`
	Badges              []Badge             `json:"badges,omitempty"`
	PrimaryBadgeID      string              `json:"primary_badge_id,omitempty"`
	ContributionScore   int                 `json:"contribution_score,omitempty"`
	ContributionSummary ContributionSummary `json:"contribution_summary,omitempty"`
}

type MetadataAdmin struct {
	Birthdate               string                  `json:"birthdate,omitempty"`
	NotificationPreferences NotificationPreferences `json:"notification_preferences,omitempty"`
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

func (m MetadataAdmin) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MetadataAdmin) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, m)
}

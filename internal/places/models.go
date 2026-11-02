package places

import (
	"encoding/json"
	"time"
)

type CollectionRecord struct {
	ID              string
	Slug            string
	Title           string
	Subtitle        *string
	Description     *string
	PreviewImageURL *string
	IconImageURL    *string
	BackgroundColor *string
	SortOrder       int
	PlacesCount     int
	Filters         []CollectionFilterRecord
}

type CollectionFilterRecord struct {
	ID          string
	Slug        string
	Title       string
	SortOrder   int
	PlacesCount int
}

type PlaceRecord struct {
	ID                       string
	Slug                     string
	Title                    string
	Kind                     KindRecord
	Summary                  *string
	DescriptionMarkdown      *string
	Status                   string
	SortOrder                int
	LocationLabel            *string
	AddressDisplayText       *string
	AddressRegion            *string
	AddressDistrict          *string
	AddressLocality          *string
	AddressStreet            *string
	AddressHouse             *string
	OpeningHoursTimezone     *string
	OpeningHoursNoteMarkdown *string
	CoverMediaID             *string
	CoverImageURL            *string
	NavigationPoint          json.RawMessage
	Geometry                 json.RawMessage
	PublishedAt              *time.Time
	CreatedAt                time.Time
	UpdatedAt                time.Time
	Gallery                  []MediaRecord
	Badges                   []BadgeRecord
	Contacts                 []ContactRecord
	Hashtags                 []HashtagRecord
	InfoBlocks               []InfoBlockRecord
	Memberships              []MembershipRecord
	RegularPeriods           []OpeningPeriodRecord
	SpecialPeriods           []SpecialOpeningPeriodRecord
}

type KindRecord struct{ ID, Key, Label string }

type MediaRecord struct {
	ID, Kind, URL string
	ThumbnailURL  *string
	Width, Height *int
	AltText       *string
	SortOrder     int
}

type BadgeRecord struct {
	ID, Title, Tone string
	SortOrder       int
}
type ContactRecord struct {
	ID, Type                string
	Label                   *string
	DisplayValue, ActionURL string
	SortOrder               int
}
type HashtagRecord struct {
	ID, Slug, Title string
	SortOrder       int
}
type InfoBlockRecord struct {
	ID, Type, IconKey, Title, ValueMarkdown   string
	BackgroundColorLight, BackgroundColorDark *string
	SortOrder                                 int
}
type MembershipRecord struct {
	ID, CollectionID  string
	SortOrder         int
	FilterMemberships []FilterMembershipRecord
}
type FilterMembershipRecord struct {
	ID, FilterID string
	SortOrder    int
}
type OpeningPeriodRecord struct {
	ID, OpenDay             string
	OpenMinutes             int
	CloseDay                string
	CloseMinutes, SortOrder int
}
type SpecialOpeningPeriodRecord struct {
	ID                        string
	StartDate                 time.Time
	EndDate                   *time.Time
	Closed                    bool
	OpenMinutes, CloseMinutes *int
	NoteMarkdown              *string
	SortOrder                 int
}

type Actor struct {
	UserID       string
	SessionID    string
	Role         string
	TokenVersion int
}

type AuditEnvelope struct {
	IP        *string
	UserAgent *string
}

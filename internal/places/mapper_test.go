package places

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMapDetailsFallsBackToNavigationPointGeometry(t *testing.T) {
	record := validRecord()
	record.Geometry = nil
	got, err := MapDetails(record)
	if err != nil {
		t.Fatalf("MapDetails() error = %v", err)
	}
	if got.Geometry.Type != "Point" || got.NavigationPoint.Type != "Point" {
		t.Fatalf("unexpected geometry: %#v / %#v", got.Geometry, got.NavigationPoint)
	}
}

func TestMapSummaryRejectsNonPointNavigationGeometry(t *testing.T) {
	record := validRecord()
	record.NavigationPoint = json.RawMessage(`{"type":"Polygon","coordinates":[]}`)
	if _, err := MapSummary(record); err == nil {
		t.Fatal("MapSummary() expected an error")
	}
}

func TestMapAdminSortsRelationsAndFormatsUTC(t *testing.T) {
	record := validRecord()
	record.CreatedAt = time.Date(2026, 10, 31, 13, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	record.UpdatedAt = record.CreatedAt.Add(time.Hour)
	record.Badges = []BadgeRecord{{ID: "b2", Title: "B", SortOrder: 2}, {ID: "b1", Title: "A", SortOrder: 1}}
	record.Memberships = []MembershipRecord{{ID: "m2", CollectionID: "c2", SortOrder: 2}, {ID: "m1", CollectionID: "c1", SortOrder: 1}}
	got, err := MapAdmin(record)
	if err != nil {
		t.Fatalf("MapAdmin() error = %v", err)
	}
	if len(got.Badges) != 2 || got.Badges[0].ID != "b1" || got.CollectionMemberships[0].CollectionID != "c1" {
		t.Fatalf("relations not sorted: %#v %#v", got.Badges, got.CollectionMemberships)
	}
	if got.CreatedAt != "2026-10-31T10:00:00Z" || got.UpdatedAt != "2026-10-31T11:00:00Z" {
		t.Fatalf("timestamps not normalized to UTC: %q %q", got.CreatedAt, got.UpdatedAt)
	}
}

func TestMapCollectionUsesEmptyFiltersArray(t *testing.T) {
	got := MapCollection(CollectionRecord{ID: "c", Slug: "nature", Title: "Nature"})
	if got.Filters == nil {
		t.Fatal("filters must be a non-nil empty slice")
	}
}

func TestMapDetailsMapsNullableAndNestedRelationsExactly(t *testing.T) {
	record := validRecord()
	record.Geometry = json.RawMessage(`{"type":"Polygon","coordinates":[[[46,43],[47,43],[46,43]]]}`)
	record.AddressLocality = stringPointer("Дубки")
	record.Gallery = []MediaRecord{{ID: "m2", Kind: "image", URL: "two", SortOrder: 2}, {ID: "m1", Kind: "image", URL: "one", SortOrder: 1}}
	record.Contacts = []ContactRecord{{ID: "c", Type: "website", DisplayValue: "example", ActionURL: "https://example.test"}}
	record.Hashtags = []HashtagRecord{{ID: "t", Slug: "canyon", Title: "Каньон"}}
	record.Badges = []BadgeRecord{{ID: "b", Title: "Popular", Tone: "accent"}}
	record.InfoBlocks = []InfoBlockRecord{
		{ID: "i2", Type: "large", IconKey: "map", Title: "Second", ValueMarkdown: "B", BackgroundColorLight: stringPointer("#ffffff"), SortOrder: 2},
		{ID: "i1", Type: "single_line", IconKey: "info", Title: "First", ValueMarkdown: "A", BackgroundColorLight: stringPointer("#ffffff"), BackgroundColorDark: stringPointer("#000000"), SortOrder: 1},
	}
	record.OpeningHoursTimezone = stringPointer("Europe/Moscow")
	record.RegularPeriods = []OpeningPeriodRecord{{ID: "r2", OpenDay: "tuesday", OpenMinutes: 600, CloseDay: "tuesday", CloseMinutes: 1080, SortOrder: 2}, {ID: "r1", OpenDay: "monday", OpenMinutes: 545, CloseDay: "monday", CloseMinutes: 1065, SortOrder: 1}}
	end := time.Date(2026, 11, 5, 12, 0, 0, 0, time.FixedZone("UTC+3", 3*60*60))
	record.SpecialPeriods = []SpecialOpeningPeriodRecord{
		{ID: "s2", StartDate: end.AddDate(0, 0, 1), Closed: true, SortOrder: 2},
		{ID: "s1", StartDate: end, EndDate: &end, OpenMinutes: intPointer(570), CloseMinutes: intPointer(1020), NoteMarkdown: stringPointer("Holiday"), SortOrder: 1},
	}

	got, err := MapDetails(record)
	if err != nil {
		t.Fatalf("MapDetails() error = %v", err)
	}
	if got.Geometry.Type != "Polygon" || got.Address == nil || got.Address.Locality == nil || *got.Address.Locality != "Дубки" {
		t.Fatalf("geometry/address=%#v %#v", got.Geometry, got.Address)
	}
	if len(got.Gallery) != 2 || got.Gallery[0].ID != "m1" || got.InfoBlocks[0].ID != "i1" || got.InfoBlocks[0].BackgroundColor == nil || got.InfoBlocks[1].BackgroundColor != nil {
		t.Fatalf("sorted/nested fields=%#v %#v", got.Gallery, got.InfoBlocks)
	}
	if got.OpeningHours == nil || got.OpeningHours.RegularPeriods[0].OpenTime != "09:05" || got.OpeningHours.RegularPeriods[0].CloseTime != "17:45" {
		t.Fatalf("regular periods=%#v", got.OpeningHours)
	}
	special := got.OpeningHours.SpecialPeriods
	if len(special) != 2 || special[0].StartDate != "2026-11-05" || special[0].EndDate == nil || *special[0].EndDate != "2026-11-05" || special[0].OpenTime == nil || *special[0].OpenTime != "09:30" || special[1].OpenTime != nil {
		t.Fatalf("special periods=%#v", special)
	}
}

func TestMapAdminKeepsRequiredCollectionsNonNilAndOptionalFieldsNull(t *testing.T) {
	record := validRecord()
	record.NavigationPoint = nil
	record.Geometry = nil
	got, err := MapAdmin(record)
	if err != nil {
		t.Fatal(err)
	}
	if got.NavigationPoint != nil || got.Geometry != nil || got.Address != nil || got.OpeningHours != nil || got.PublishedAt != nil {
		t.Fatalf("optional fields must remain nil: %#v", got)
	}
	if got.Gallery == nil || got.Contacts == nil || got.Badges == nil || got.Hashtags == nil || got.InfoBlocks == nil || got.CollectionMemberships == nil {
		t.Fatalf("required collections must encode as []: %#v", got)
	}
}

func TestGeometryDecodersRejectMalformedAndUnsupportedValues(t *testing.T) {
	for name, raw := range map[string]json.RawMessage{
		"malformed point":      json.RawMessage(`{"type":`),
		"unsupported geometry": json.RawMessage(`{"type":"LineString","coordinates":[]}`),
	} {
		t.Run(name, func(t *testing.T) {
			record := validRecord()
			if name == "malformed point" {
				record.NavigationPoint = raw
			} else {
				record.Geometry = raw
			}
			if _, err := MapDetails(record); err == nil {
				t.Fatal("expected mapping error")
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
func intPointer(value int) *int          { return &value }

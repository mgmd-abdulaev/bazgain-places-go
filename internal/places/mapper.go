package places

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/mgmd-abdulaev/bazgain-places-go/internal/wire"
)

func MapCollection(row CollectionRecord) wire.PlaceCollectionSummary {
	filters := make([]wire.PlaceCollectionFilterSummary, 0, len(row.Filters))
	for _, filter := range row.Filters {
		count := filter.PlacesCount
		filters = append(filters, wire.PlaceCollectionFilterSummary{ID: filter.ID, Slug: filter.Slug, Title: filter.Title, SortOrder: filter.SortOrder, PlacesCount: &count})
	}
	count := row.PlacesCount
	return wire.PlaceCollectionSummary{ID: row.ID, Slug: row.Slug, Title: row.Title, Subtitle: row.Subtitle, Description: row.Description, PreviewImageURL: row.PreviewImageURL, IconImageURL: row.IconImageURL, BackgroundColor: row.BackgroundColor, SortOrder: row.SortOrder, PlacesCount: &count, Filters: filters}
}

func MapSummary(row PlaceRecord) (wire.PlaceSummary, error) {
	point, err := decodePoint(row.NavigationPoint)
	if err != nil {
		return wire.PlaceSummary{}, err
	}
	return wire.PlaceSummary{
		ID: row.ID, Slug: row.Slug, Title: row.Title,
		Kind:    wire.PlaceKind{Key: row.Kind.Key, Label: row.Kind.Label},
		Summary: row.Summary, SortOrder: row.SortOrder,
		CoverImageURL: stringOrEmpty(row.CoverImageURL), LocationLabel: row.LocationLabel,
		NavigationPoint: point, Badges: publicBadges(row.Badges), Hashtags: publicTags(row.Hashtags),
		CollectionMemberships: memberships(row.Memberships), OpeningHours: openingHours(row),
	}, nil
}

func MapDetails(row PlaceRecord) (wire.PlaceDetails, error) {
	point, err := decodePoint(row.NavigationPoint)
	if err != nil {
		return wire.PlaceDetails{}, err
	}
	geometryRaw := row.Geometry
	if len(geometryRaw) == 0 || string(geometryRaw) == "null" {
		geometryRaw = row.NavigationPoint
	}
	geometry, err := decodeGeometry(geometryRaw)
	if err != nil {
		return wire.PlaceDetails{}, err
	}
	return wire.PlaceDetails{
		ID: row.ID, Slug: row.Slug, Title: row.Title, Kind: wire.PlaceKind{Key: row.Kind.Key, Label: row.Kind.Label},
		Summary: row.Summary, DescriptionMarkdown: row.DescriptionMarkdown,
		Gallery: publicMedia(row.Gallery), Geometry: geometry, NavigationPoint: point,
		Address: address(row), Contacts: publicContacts(row.Contacts), Hashtags: publicTags(row.Hashtags),
		Badges: publicBadges(row.Badges), InfoBlocks: infoBlocks(row.InfoBlocks), OpeningHours: openingHours(row),
	}, nil
}

func MapAdmin(row PlaceRecord) (wire.PlaceAdminView, error) {
	var point *wire.GeoJSONPoint
	if len(row.NavigationPoint) > 0 && string(row.NavigationPoint) != "null" {
		decoded, err := decodePoint(row.NavigationPoint)
		if err != nil {
			return wire.PlaceAdminView{}, err
		}
		point = &decoded
	}
	var geometry *wire.GeoJSONGeometry
	if len(row.Geometry) > 0 && string(row.Geometry) != "null" {
		decoded, err := decodeGeometry(row.Geometry)
		if err != nil {
			return wire.PlaceAdminView{}, err
		}
		geometry = &decoded
	}
	gallery := make([]wire.PlaceMedia, 0, len(row.Gallery))
	for _, item := range sortedMedia(row.Gallery) {
		gallery = append(gallery, wire.PlaceMedia{ID: item.ID, Kind: item.Kind, URL: item.URL, ThumbnailURL: item.ThumbnailURL, Width: item.Width, Height: item.Height, AltText: item.AltText, SortOrder: item.SortOrder})
	}
	badges := make([]wire.PlaceBadge, 0, len(row.Badges))
	for _, item := range sortedBadges(row.Badges) {
		badges = append(badges, wire.PlaceBadge{ID: item.ID, Title: item.Title, Tone: item.Tone, SortOrder: item.SortOrder})
	}
	contacts := make([]wire.PlaceContactAdmin, 0, len(row.Contacts))
	for _, item := range sortedContacts(row.Contacts) {
		contacts = append(contacts, wire.PlaceContactAdmin{ID: item.ID, Type: item.Type, Label: item.Label, DisplayValue: item.DisplayValue, ActionURL: item.ActionURL, SortOrder: item.SortOrder})
	}
	hashtags := make([]wire.PlaceHashtag, 0, len(row.Hashtags))
	for _, item := range sortedTags(row.Hashtags) {
		hashtags = append(hashtags, wire.PlaceHashtag{ID: item.ID, Slug: item.Slug, Title: item.Title, SortOrder: item.SortOrder})
	}
	return wire.PlaceAdminView{
		ID: row.ID, Slug: row.Slug, Title: row.Title, Kind: wire.PlaceKindRef{ID: row.Kind.ID, Key: row.Kind.Key, Label: row.Kind.Label},
		Summary: row.Summary, DescriptionMarkdown: row.DescriptionMarkdown, Status: row.Status, SortOrder: row.SortOrder,
		LocationLabel: row.LocationLabel, Address: address(row), OpeningHours: openingHours(row), NavigationPoint: point, Geometry: geometry,
		Gallery: gallery, Contacts: contacts, Badges: badges, Hashtags: hashtags, InfoBlocks: infoBlocks(row.InfoBlocks), CollectionMemberships: memberships(row.Memberships),
		CoverMediaID: row.CoverMediaID, CoverImageURL: row.CoverImageURL,
		PublishedAt: formatOptionalTime(row.PublishedAt), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt),
	}, nil
}

func publicMedia(rows []MediaRecord) []wire.MediaAsset {
	out := make([]wire.MediaAsset, 0, len(rows))
	for _, r := range sortedMedia(rows) {
		out = append(out, wire.MediaAsset{ID: r.ID, Type: r.Kind, URL: r.URL, ThumbnailURL: r.ThumbnailURL, Width: r.Width, Height: r.Height, AltText: r.AltText})
	}
	return out
}
func publicBadges(rows []BadgeRecord) []wire.Badge {
	out := make([]wire.Badge, 0, len(rows))
	for _, r := range sortedBadges(rows) {
		out = append(out, wire.Badge{ID: r.ID, Title: r.Title, Tone: r.Tone})
	}
	return out
}
func publicTags(rows []HashtagRecord) []wire.PlaceTag {
	out := make([]wire.PlaceTag, 0, len(rows))
	for _, r := range sortedTags(rows) {
		out = append(out, wire.PlaceTag{ID: r.ID, Slug: r.Slug, Title: r.Title})
	}
	return out
}
func publicContacts(rows []ContactRecord) []wire.PlaceContact {
	out := make([]wire.PlaceContact, 0, len(rows))
	for _, r := range sortedContacts(rows) {
		out = append(out, wire.PlaceContact{ID: r.ID, Type: r.Type, Label: r.Label, DisplayValue: r.DisplayValue, ActionURL: r.ActionURL})
	}
	return out
}
func infoBlocks(rows []InfoBlockRecord) []wire.PlaceInfoBlock {
	sorted := append([]InfoBlockRecord(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].SortOrder == sorted[j].SortOrder {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].SortOrder < sorted[j].SortOrder
	})
	out := make([]wire.PlaceInfoBlock, 0, len(sorted))
	for _, r := range sorted {
		var bg *wire.ThemedColor
		if r.BackgroundColorLight != nil && r.BackgroundColorDark != nil {
			bg = &wire.ThemedColor{Light: *r.BackgroundColorLight, Dark: *r.BackgroundColorDark}
		}
		out = append(out, wire.PlaceInfoBlock{ID: r.ID, Type: r.Type, IconKey: r.IconKey, Title: r.Title, ValueMarkdown: r.ValueMarkdown, BackgroundColor: bg, SortOrder: r.SortOrder})
	}
	return out
}

func memberships(rows []MembershipRecord) []wire.PlaceCollectionMembership {
	sorted := append([]MembershipRecord(nil), rows...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].SortOrder == sorted[j].SortOrder {
			return sorted[i].ID < sorted[j].ID
		}
		return sorted[i].SortOrder < sorted[j].SortOrder
	})
	out := make([]wire.PlaceCollectionMembership, 0, len(sorted))
	for _, r := range sorted {
		filters := append([]FilterMembershipRecord(nil), r.FilterMemberships...)
		sort.Slice(filters, func(i, j int) bool {
			if filters[i].SortOrder == filters[j].SortOrder {
				return filters[i].ID < filters[j].ID
			}
			return filters[i].SortOrder < filters[j].SortOrder
		})
		fm := make([]wire.PlaceCollectionFilterMembership, 0, len(filters))
		for _, f := range filters {
			fm = append(fm, wire.PlaceCollectionFilterMembership{FilterID: f.FilterID, SortOrder: f.SortOrder})
		}
		out = append(out, wire.PlaceCollectionMembership{CollectionID: r.CollectionID, SortOrder: r.SortOrder, FilterMemberships: fm})
	}
	return out
}

func openingHours(row PlaceRecord) *wire.PlaceOpeningHours {
	if row.OpeningHoursTimezone == nil && len(row.RegularPeriods) == 0 && len(row.SpecialPeriods) == 0 {
		return nil
	}
	regular := append([]OpeningPeriodRecord(nil), row.RegularPeriods...)
	sort.Slice(regular, func(i, j int) bool {
		if regular[i].SortOrder == regular[j].SortOrder {
			return regular[i].ID < regular[j].ID
		}
		return regular[i].SortOrder < regular[j].SortOrder
	})
	rp := make([]wire.PlaceOpeningHoursPeriod, 0, len(regular))
	for _, p := range regular {
		rp = append(rp, wire.PlaceOpeningHoursPeriod{OpenDay: p.OpenDay, OpenTime: minutes(p.OpenMinutes), CloseDay: p.CloseDay, CloseTime: minutes(p.CloseMinutes)})
	}
	special := append([]SpecialOpeningPeriodRecord(nil), row.SpecialPeriods...)
	sort.Slice(special, func(i, j int) bool {
		if special[i].SortOrder == special[j].SortOrder {
			return special[i].ID < special[j].ID
		}
		return special[i].SortOrder < special[j].SortOrder
	})
	sp := make([]wire.PlaceSpecialOpeningHoursPeriod, 0, len(special))
	for _, p := range special {
		var end *string
		if p.EndDate != nil {
			v := p.EndDate.UTC().Format("2006-01-02")
			end = &v
		}
		var open, close *string
		if p.OpenMinutes != nil {
			v := minutes(*p.OpenMinutes)
			open = &v
		}
		if p.CloseMinutes != nil {
			v := minutes(*p.CloseMinutes)
			close = &v
		}
		sp = append(sp, wire.PlaceSpecialOpeningHoursPeriod{StartDate: p.StartDate.UTC().Format("2006-01-02"), EndDate: end, Closed: p.Closed, OpenTime: open, CloseTime: close, NoteMarkdown: p.NoteMarkdown})
	}
	tz := ""
	if row.OpeningHoursTimezone != nil {
		tz = *row.OpeningHoursTimezone
	}
	return &wire.PlaceOpeningHours{Timezone: tz, NoteMarkdown: row.OpeningHoursNoteMarkdown, RegularPeriods: rp, SpecialPeriods: sp}
}

func address(row PlaceRecord) *wire.PlaceAddress {
	if row.AddressDisplayText == nil && row.AddressRegion == nil && row.AddressDistrict == nil && row.AddressLocality == nil && row.AddressStreet == nil && row.AddressHouse == nil {
		return nil
	}
	return &wire.PlaceAddress{DisplayText: row.AddressDisplayText, Region: row.AddressRegion, District: row.AddressDistrict, Locality: row.AddressLocality, Street: row.AddressStreet, House: row.AddressHouse}
}
func decodePoint(raw json.RawMessage) (wire.GeoJSONPoint, error) {
	var p wire.GeoJSONPoint
	if err := json.Unmarshal(raw, &p); err != nil {
		return p, fmt.Errorf("decode GeoJSON point: %w", err)
	}
	if p.Type != "Point" {
		return p, fmt.Errorf("expected Point, got %q", p.Type)
	}
	return p, nil
}
func decodeGeometry(raw json.RawMessage) (wire.GeoJSONGeometry, error) {
	var g wire.GeoJSONGeometry
	if err := json.Unmarshal(raw, &g); err != nil {
		return g, fmt.Errorf("decode GeoJSON geometry: %w", err)
	}
	if g.Type != "Point" && g.Type != "Polygon" && g.Type != "MultiPolygon" {
		return g, fmt.Errorf("unsupported geometry %q", g.Type)
	}
	return g, nil
}
func minutes(v int) string { return fmt.Sprintf("%02d:%02d", v/60, v%60) }
func stringOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func formatTime(v time.Time) string { return v.UTC().Format(time.RFC3339Nano) }
func formatOptionalTime(v *time.Time) *string {
	if v == nil {
		return nil
	}
	s := formatTime(*v)
	return &s
}
func sortedMedia(in []MediaRecord) []MediaRecord {
	out := append([]MediaRecord(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].ID < out[j].ID
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}
func sortedBadges(in []BadgeRecord) []BadgeRecord {
	out := append([]BadgeRecord(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].ID < out[j].ID
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}
func sortedContacts(in []ContactRecord) []ContactRecord {
	out := append([]ContactRecord(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].ID < out[j].ID
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}
func sortedTags(in []HashtagRecord) []HashtagRecord {
	out := append([]HashtagRecord(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].ID < out[j].ID
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}

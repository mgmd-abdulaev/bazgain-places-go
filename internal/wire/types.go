package wire

import "encoding/json"

type PlaceCollectionsResponse struct {
	Collections []PlaceCollectionSummary `json:"collections"`
}

type PlaceCollectionSummary struct {
	ID              string                         `json:"id"`
	Slug            string                         `json:"slug"`
	Title           string                         `json:"title"`
	Subtitle        *string                        `json:"subtitle"`
	Description     *string                        `json:"description"`
	PreviewImageURL *string                        `json:"preview_image_url"`
	IconImageURL    *string                        `json:"icon_image_url"`
	BackgroundColor *string                        `json:"background_color"`
	SortOrder       int                            `json:"sort_order"`
	PlacesCount     *int                           `json:"places_count"`
	Filters         []PlaceCollectionFilterSummary `json:"filters"`
}

type PlaceCollectionFilterSummary struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	SortOrder   int    `json:"sort_order"`
	PlacesCount *int   `json:"places_count"`
}

type PlaceListResponse struct {
	Items []PlaceSummary `json:"items"`
}

type PlaceDetailsResponse struct {
	Place PlaceDetails `json:"place"`
}

type PlaceKind struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type PlaceKindRef struct {
	ID    string `json:"id"`
	Key   string `json:"key"`
	Label string `json:"label"`
}

type GeoJSONPoint struct {
	Type        string     `json:"type"`
	Coordinates [2]float64 `json:"coordinates"`
}

type GeoJSONGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type PlaceSummary struct {
	ID                    string                      `json:"id"`
	Slug                  string                      `json:"slug"`
	Title                 string                      `json:"title"`
	Kind                  PlaceKind                   `json:"kind"`
	Summary               *string                     `json:"summary"`
	SortOrder             int                         `json:"sort_order"`
	CoverImageURL         string                      `json:"cover_image_url"`
	LocationLabel         *string                     `json:"location_label"`
	NavigationPoint       GeoJSONPoint                `json:"navigation_point"`
	Badges                []Badge                     `json:"badges"`
	Hashtags              []PlaceTag                  `json:"hashtags"`
	CollectionMemberships []PlaceCollectionMembership `json:"collection_memberships"`
	OpeningHours          *PlaceOpeningHours          `json:"opening_hours"`
}

type PlaceDetails struct {
	ID                  string             `json:"id"`
	Slug                string             `json:"slug"`
	Title               string             `json:"title"`
	Kind                PlaceKind          `json:"kind"`
	Summary             *string            `json:"summary"`
	DescriptionMarkdown *string            `json:"description_markdown"`
	Gallery             []MediaAsset       `json:"gallery"`
	Geometry            GeoJSONGeometry    `json:"geometry"`
	NavigationPoint     GeoJSONPoint       `json:"navigation_point"`
	Address             *PlaceAddress      `json:"address"`
	Contacts            []PlaceContact     `json:"contacts"`
	Hashtags            []PlaceTag         `json:"hashtags"`
	Badges              []Badge            `json:"badges"`
	InfoBlocks          []PlaceInfoBlock   `json:"info_blocks"`
	OpeningHours        *PlaceOpeningHours `json:"opening_hours"`
}

type MediaAsset struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnail_url"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
	AltText      *string `json:"alt_text"`
}

type PlaceMedia struct {
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	URL          string  `json:"url"`
	ThumbnailURL *string `json:"thumbnail_url"`
	Width        *int    `json:"width"`
	Height       *int    `json:"height"`
	AltText      *string `json:"alt_text"`
	SortOrder    int     `json:"sort_order"`
}

type Badge struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Tone  string `json:"tone"`
}

type PlaceBadge struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Tone      string `json:"tone"`
	SortOrder int    `json:"sort_order"`
}

type PlaceTag struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type PlaceHashtag struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	SortOrder int    `json:"sort_order"`
}

type PlaceContact struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Label        *string `json:"label"`
	DisplayValue string  `json:"display_value"`
	ActionURL    string  `json:"action_url"`
}

type PlaceContactAdmin struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Label        *string `json:"label"`
	DisplayValue string  `json:"display_value"`
	ActionURL    string  `json:"action_url"`
	SortOrder    int     `json:"sort_order"`
}

type ThemedColor struct {
	Light string `json:"light"`
	Dark  string `json:"dark"`
}

type PlaceInfoBlock struct {
	ID              string       `json:"id"`
	Type            string       `json:"type"`
	IconKey         string       `json:"icon_key"`
	Title           string       `json:"title"`
	ValueMarkdown   string       `json:"value_markdown"`
	BackgroundColor *ThemedColor `json:"background_color"`
	SortOrder       int          `json:"sort_order"`
}

type PlaceAddress struct {
	DisplayText *string `json:"display_text"`
	Region      *string `json:"region"`
	District    *string `json:"district"`
	Locality    *string `json:"locality"`
	Street      *string `json:"street"`
	House       *string `json:"house"`
}

type PlaceOpeningHours struct {
	Timezone       string                           `json:"timezone"`
	NoteMarkdown   *string                          `json:"note_markdown"`
	RegularPeriods []PlaceOpeningHoursPeriod        `json:"regular_periods"`
	SpecialPeriods []PlaceSpecialOpeningHoursPeriod `json:"special_periods"`
}

type PlaceOpeningHoursPeriod struct {
	OpenDay   string `json:"open_day"`
	OpenTime  string `json:"open_time"`
	CloseDay  string `json:"close_day"`
	CloseTime string `json:"close_time"`
}

type PlaceSpecialOpeningHoursPeriod struct {
	StartDate    string  `json:"start_date"`
	EndDate      *string `json:"end_date"`
	Closed       bool    `json:"closed"`
	OpenTime     *string `json:"open_time"`
	CloseTime    *string `json:"close_time"`
	NoteMarkdown *string `json:"note_markdown"`
}

type PlaceCollectionMembership struct {
	CollectionID      string                            `json:"collection_id"`
	SortOrder         int                               `json:"sort_order"`
	FilterMemberships []PlaceCollectionFilterMembership `json:"filter_memberships"`
}

type PlaceCollectionFilterMembership struct {
	FilterID  string `json:"filter_id"`
	SortOrder int    `json:"sort_order"`
}

type PlaceAdminView struct {
	ID                    string                      `json:"id"`
	Slug                  string                      `json:"slug"`
	Title                 string                      `json:"title"`
	Kind                  PlaceKindRef                `json:"kind"`
	Summary               *string                     `json:"summary"`
	DescriptionMarkdown   *string                     `json:"description_markdown"`
	Status                string                      `json:"status"`
	SortOrder             int                         `json:"sort_order"`
	LocationLabel         *string                     `json:"location_label"`
	Address               *PlaceAddress               `json:"address"`
	OpeningHours          *PlaceOpeningHours          `json:"opening_hours"`
	NavigationPoint       *GeoJSONPoint               `json:"navigation_point"`
	Geometry              *GeoJSONGeometry            `json:"geometry"`
	Gallery               []PlaceMedia                `json:"gallery"`
	Contacts              []PlaceContactAdmin         `json:"contacts"`
	Badges                []PlaceBadge                `json:"badges"`
	Hashtags              []PlaceHashtag              `json:"hashtags"`
	InfoBlocks            []PlaceInfoBlock            `json:"info_blocks"`
	CollectionMemberships []PlaceCollectionMembership `json:"collection_memberships"`
	CoverMediaID          *string                     `json:"cover_media_id"`
	CoverImageURL         *string                     `json:"cover_image_url"`
	PublishedAt           *string                     `json:"published_at"`
	CreatedAt             string                      `json:"created_at"`
	UpdatedAt             string                      `json:"updated_at"`
}

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorBody struct {
	Code    string       `json:"code"`
	Message string       `json:"message"`
	Fields  []FieldError `json:"fields,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorBody `json:"error"`
}

package contract_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func loadSpec(t *testing.T) *openapi3.T {
	t.Helper()
	_, source, _, _ := runtime.Caller(0)
	path := filepath.Join(filepath.Dir(source), "..", "..", "api", "openapi.yaml")
	document, err := openapi3.NewLoader().LoadFromFile(path)
	if err != nil {
		t.Fatalf("load OpenAPI document: %v", err)
	}
	if err := document.Validate(context.Background(), openapi3.EnableSchemaFormatValidation()); err != nil {
		t.Fatalf("validate OpenAPI document: %v", err)
	}
	return document
}

func responseSchema(t *testing.T, document *openapi3.T, path, method string, status int) *openapi3.Schema {
	t.Helper()
	pathItem := document.Paths.Find(path)
	if pathItem == nil {
		t.Fatalf("path %q is not documented", path)
	}
	operation := pathItem.GetOperation(method)
	if operation == nil {
		t.Fatalf("method %s %s is not documented", method, path)
	}
	response := operation.Responses.Status(status)
	if response == nil || response.Value == nil {
		t.Fatalf("response %d for %s %s is not documented", status, method, path)
	}
	media := response.Value.Content.Get("application/json")
	if media == nil || media.Schema == nil || media.Schema.Value == nil {
		t.Fatalf("response %d for %s %s has no JSON schema", status, method, path)
	}
	return media.Schema.Value
}

func decodeJSON(t *testing.T, raw string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("decode test payload: %v", err)
	}
	return value
}

func TestFiveEndpointSuccessPayloadsConformToOpenAPI(t *testing.T) {
	document := loadSpec(t)
	admin := `{"id":"019a3ef2-4d76-7d9d-8c25-7b51a117c654","slug":"sulak","title":"Sulak","kind":{"id":"kind-1","key":"nature","label":"Nature"},"summary":null,"description_markdown":null,"status":"published","sort_order":1,"location_label":null,"address":null,"opening_hours":null,"navigation_point":{"type":"Point","coordinates":[46.8,43.0]},"geometry":{"type":"Polygon","coordinates":[[[46.8,43.0],[46.9,43.0],[46.8,43.0]]]},"gallery":[],"contacts":[],"badges":[],"hashtags":[],"info_blocks":[],"collection_memberships":[],"cover_media_id":null,"cover_image_url":null,"published_at":"2026-10-31T10:00:00Z","created_at":"2026-10-31T09:00:00Z","updated_at":"2026-10-31T10:00:00Z"}`
	tests := []struct {
		name, path, method, payload string
	}{
		{"collections", "/v1/place-collections", http.MethodGet, `{"collections":[{"id":"collection-1","slug":"top","title":"Top","sort_order":1,"filters":[]}]}`},
		{"places", "/v1/places", http.MethodGet, `{"items":[{"id":"place-1","slug":"sulak","title":"Sulak","kind":{"key":"nature","label":"Nature"},"sort_order":1,"cover_image_url":"https://cdn.example/place.jpg","navigation_point":{"type":"Point","coordinates":[46.8,43.0]},"badges":[],"hashtags":[],"collection_memberships":[]}]}`},
		{"details", "/v1/places/{placeSlug}", http.MethodGet, `{"place":{"id":"place-1","slug":"sulak","title":"Sulak","kind":{"key":"nature","label":"Nature"},"gallery":[{"id":"media-1","type":"image","url":"https://cdn.example/place.jpg"}],"geometry":{"type":"MultiPolygon","coordinates":[[[[46.8,43.0],[46.9,43.0],[46.8,43.0]]]]},"navigation_point":{"type":"Point","coordinates":[46.8,43.0]},"contacts":[],"hashtags":[],"badges":[],"info_blocks":[]}}`},
		{"publish", "/v1/admin/places/{id}/publish", http.MethodPost, admin},
		{"archive", "/v1/admin/places/{id}/archive", http.MethodPost, admin},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema := responseSchema(t, document, test.path, test.method, http.StatusOK)
			if err := schema.VisitJSON(decodeJSON(t, test.payload), openapi3.VisitAsResponse(), openapi3.EnableJSONSchema2020(), openapi3.EnableFormatValidation()); err != nil {
				t.Fatalf("payload does not conform: %v", err)
			}
		})
	}
}

func TestFiveEndpointErrorPayloadsConformToOpenAPI(t *testing.T) {
	document := loadSpec(t)
	tests := []struct {
		name, path, method string
		status             int
	}{
		{"collections", "/v1/place-collections", http.MethodGet, http.StatusTooManyRequests},
		{"places", "/v1/places", http.MethodGet, http.StatusTooManyRequests},
		{"details", "/v1/places/{placeSlug}", http.MethodGet, http.StatusNotFound},
		{"publish", "/v1/admin/places/{id}/publish", http.MethodPost, http.StatusConflict},
		{"archive", "/v1/admin/places/{id}/archive", http.MethodPost, http.StatusUnprocessableEntity},
	}
	payload := decodeJSON(t, `{"error":{"code":"place_not_found","message":"Place not found","fields":[{"field":"id","code":"invalid_uuid","message":"Must be a UUID"}]}}`)
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			schema := responseSchema(t, document, test.path, test.method, test.status)
			if err := schema.VisitJSON(payload, openapi3.VisitAsResponse(), openapi3.EnableJSONSchema2020()); err != nil {
				t.Fatalf("error payload does not conform: %v", err)
			}
		})
	}
}

func TestLockedGeometryAndGalleryConstraintsRejectInvalidPayloads(t *testing.T) {
	document := loadSpec(t)
	detail := responseSchema(t, document, "/v1/places/{placeSlug}", http.MethodGet, http.StatusOK)
	invalid := []string{
		`{"place":{"id":"p","slug":"p","title":"P","kind":{"key":"nature","label":"Nature"},"gallery":[],"geometry":{"type":"Point","coordinates":[46.8,43.0]},"navigation_point":{"type":"Point","coordinates":[46.8,43.0]},"contacts":[],"hashtags":[],"badges":[],"info_blocks":[]}}`,
		`{"place":{"id":"p","slug":"p","title":"P","kind":{"key":"nature","label":"Nature"},"gallery":[{"id":"m","type":"image","url":"https://cdn.example/p.jpg"}],"geometry":{"type":"Polygon","coordinates":[[[46.8],[46.9,43.0]]]},"navigation_point":{"type":"Point","coordinates":[46.8,43.0]},"contacts":[],"hashtags":[],"badges":[],"info_blocks":[]}}`,
	}
	for i, raw := range invalid {
		if err := detail.VisitJSON(decodeJSON(t, raw), openapi3.VisitAsResponse(), openapi3.EnableJSONSchema2020()); err == nil {
			t.Fatalf("invalid payload %d unexpectedly conformed", i)
		}
	}
}

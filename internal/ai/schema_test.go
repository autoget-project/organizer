package ai_test

import (
	"encoding/json"
	"testing"

	"organizer/internal/ai"
)

type NestedStruct struct {
	InnerField string `json:"inner_field" description:"An inner field"`
	InnerVal   int    `json:"inner_val"`
}

type SampleStruct struct {
	Title       string         `json:"title" description:"The title of the item"`
	Season      int            `json:"season"`
	IsAnim      bool           `json:"is_anim"`
	Tags        []string       `json:"tags"`
	Category    string         `json:"category" enum:"movie,tv_series,bango_porn"`
	Nested      NestedStruct   `json:"nested"`
	NestedSlice []NestedStruct `json:"nested_slice"`
	Ignored     string         `json:"-"`
	Optional    string         `json:"optional,omitempty"`
}

func TestGenerateStrictJSONSchema(t *testing.T) {
	schema, err := ai.GenerateStrictJSONSchema(SampleStruct{})
	if err != nil {
		t.Fatalf("GenerateStrictJSONSchema failed: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("expected schema.Type object, got %s", schema.Type)
	}

	if schema.AdditionalProperties != false {
		t.Errorf("expected additionalProperties to be false in strict mode")
	}

	// Verify required fields include all exported non-ignored fields in strict mode
	expectedRequired := map[string]bool{
		"title":        true,
		"season":       true,
		"is_anim":      true,
		"tags":         true,
		"category":     true,
		"nested":       true,
		"nested_slice": true,
		"optional":     true,
	}

	for _, req := range schema.Required {
		delete(expectedRequired, req)
	}
	if len(expectedRequired) > 0 {
		t.Errorf("missing required fields in strict schema: %v", expectedRequired)
	}

	// Verify properties
	titleProp, ok := schema.Properties["title"]
	if !ok || titleProp.Type != "string" || titleProp.Description != "The title of the item" {
		t.Errorf("invalid title property: %+v", titleProp)
	}

	catProp, ok := schema.Properties["category"]
	if !ok || len(catProp.Enum) != 3 {
		t.Errorf("invalid category enum property: %+v", catProp)
	}

	// Verify nested struct in strict mode
	nestedProp, ok := schema.Properties["nested"]
	if !ok || nestedProp.Type != "object" || nestedProp.AdditionalProperties != false {
		t.Errorf("invalid nested property: %+v", nestedProp)
	}
	if len(nestedProp.Required) != 2 {
		t.Errorf("expected 2 required fields in nested struct, got %d", len(nestedProp.Required))
	}

	// Verify marshalable to JSON
	bytes, err := json.Marshal(schema)
	if err != nil {
		t.Fatalf("failed to marshal JSONSchema: %v", err)
	}
	if len(bytes) == 0 {
		t.Errorf("marshaled json is empty")
	}
}

func TestGenerateOpenAPISchema(t *testing.T) {
	schema, err := ai.GenerateOpenAPISchema(SampleStruct{})
	if err != nil {
		t.Fatalf("GenerateOpenAPISchema failed: %v", err)
	}

	if schema.Type != "object" {
		t.Errorf("expected schema.Type object, got %s", schema.Type)
	}

	if schema.AdditionalProperties != true {
		t.Errorf("expected additionalProperties to be true in OpenAPI mode")
	}

	// Optional field has omitempty, so it should not be in required
	for _, req := range schema.Required {
		if req == "optional" {
			t.Errorf("field with omitempty should not be required in OpenAPI mode")
		}
	}
}

func TestGenerateSchema_Nil(t *testing.T) {
	_, err := ai.GenerateStrictJSONSchema(nil)
	if err == nil {
		t.Errorf("expected error for nil, got nil")
	}
}

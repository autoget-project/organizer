package ai_test

import (
	"encoding/json"
	"testing"
	"time"

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
	// Surplus required entries (e.g. a json:"-" field leaking in) must also fail.
	actualRequired := make(map[string]bool, len(schema.Required))
	for _, req := range schema.Required {
		if _, dup := actualRequired[req]; dup {
			t.Errorf("duplicate required field in strict schema: %s", req)
		}
		actualRequired[req] = true
	}
	for _, req := range schema.Required {
		switch req {
		case "title", "season", "is_anim", "tags", "category", "nested", "nested_slice", "optional":
		default:
			t.Errorf("unexpected required field in strict schema: %s", req)
		}
	}
	if _, exists := schema.Properties["ignored"]; exists {
		t.Errorf("json:\"-\" field should not appear in properties")
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

	if schema.AdditionalProperties != nil {
		t.Errorf("expected additionalProperties to be omitted in OpenAPI mode, got %v", schema.AdditionalProperties)
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

type MapTimeStruct struct {
	Meta   map[string]string      `json:"meta"`
	Any    map[string]interface{} `json:"any"`
	When   time.Time              `json:"when"`
	Points []map[string]int       `json:"points"`
}

func TestGenerateSchema_MapsAndTime(t *testing.T) {
	schema, err := ai.GenerateStrictJSONSchema(MapTimeStruct{})
	if err != nil {
		t.Fatalf("GenerateStrictJSONSchema failed: %v", err)
	}

	metaProp, ok := schema.Properties["meta"]
	if !ok || metaProp.Type != "object" {
		t.Fatalf("invalid meta property: %+v", metaProp)
	}
	metaVal, ok := metaProp.AdditionalProperties.(ai.JSONSchema)
	if !ok || metaVal.Type != "string" {
		t.Errorf("expected meta additionalProperties {type:string}, got %v", metaProp.AdditionalProperties)
	}

	anyProp, ok := schema.Properties["any"]
	if !ok || anyProp.AdditionalProperties != true {
		t.Errorf("expected any additionalProperties true, got %v", anyProp.AdditionalProperties)
	}

	whenProp, ok := schema.Properties["when"]
	if !ok || whenProp.Type != "string" {
		t.Errorf("expected time.Time mapped to {type:string}, got %+v", whenProp)
	}

	pointsProp, ok := schema.Properties["points"]
	if !ok || pointsProp.Type != "array" {
		t.Fatalf("invalid points property: %+v", pointsProp)
	}
	if pointsProp.Items == nil || pointsProp.Items.AdditionalProperties == nil {
		t.Errorf("expected points items to carry typed additionalProperties, got %+v", pointsProp.Items)
	}
}

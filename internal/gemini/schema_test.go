package gemini

import (
	"testing"
)

func TestSanitizeSchema_Nil(t *testing.T) {
	// sanitizeImpl(nil) returns nil; SanitizeSchema wraps it and
	// falls back to returning the original (nil) schema.
	// A nil map[string]any is returned, which is the original input.
	result := SanitizeSchema(nil)
	if len(result) != 0 {
		t.Errorf("SanitizeSchema(nil) len = %d, want 0", len(result))
	}
}

func TestSanitizeSchema_EmptyMap(t *testing.T) {
	result := SanitizeSchema(map[string]any{})
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestSanitizeSchema_RemovesAdditionalProperties(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"additionalProperties": false,
	}
	result := SanitizeSchema(schema)
	if _, ok := result["additionalProperties"]; ok {
		t.Error("additionalProperties should be removed")
	}
	if result["type"] != "object" {
		t.Errorf("type = %v", result["type"])
	}
}

func TestSanitizeSchema_EnumIntegerToString(t *testing.T) {
	schema := map[string]any{
		"type": "integer",
		"enum": []any{1, 2, 3},
	}
	result := SanitizeSchema(schema)
	if result["type"] != "string" {
		t.Errorf("type = %v, want string", result["type"])
	}
	enumArr := result["enum"].([]any)
	if enumArr[0] != "1" || enumArr[1] != "2" || enumArr[2] != "3" {
		t.Errorf("enum = %v, want [1 2 3] as strings", enumArr)
	}
}

func TestSanitizeSchema_EnumNumberToString(t *testing.T) {
	schema := map[string]any{
		"type": "number",
		"enum": []any{1.5, 2.5},
	}
	result := SanitizeSchema(schema)
	if result["type"] != "string" {
		t.Errorf("type = %v, want string", result["type"])
	}
}

func TestSanitizeSchema_EnumStringNoChange(t *testing.T) {
	schema := map[string]any{
		"type": "string",
		"enum": []any{"a", "b", "c"},
	}
	result := SanitizeSchema(schema)
	if result["type"] != "string" {
		t.Errorf("type = %v, want string", result["type"])
	}
	enumArr := result["enum"].([]any)
	if enumArr[0] != "a" {
		t.Errorf("enum[0] = %v", enumArr[0])
	}
}

func TestSanitizeSchema_FilterRequiredFields(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name", "nonexistent"},
	}
	result := SanitizeSchema(schema)
	required := result["required"].([]any)
	if len(required) != 1 {
		t.Fatalf("required = %v, want [name]", required)
	}
	if required[0] != "name" {
		t.Errorf("required[0] = %v", required[0])
	}
}

func TestSanitizeSchema_ArrayItemsNoType(t *testing.T) {
	schema := map[string]any{
		"type":  "array",
		"items": map[string]any{},
	}
	result := SanitizeSchema(schema)
	items := result["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf("items.type = %v, want string", items["type"])
	}
}

func TestSanitizeSchema_ArrayItemsNil(t *testing.T) {
	schema := map[string]any{
		"type": "array",
	}
	result := SanitizeSchema(schema)
	items := result["items"].(map[string]any)
	if items == nil {
		t.Fatal("items should be set")
	}
}

func TestSanitizeSchema_ArrayItemsWithType(t *testing.T) {
	schema := map[string]any{
		"type":  "array",
		"items": map[string]any{"type": "integer"},
	}
	result := SanitizeSchema(schema)
	items := result["items"].(map[string]any)
	if items["type"] != "integer" {
		t.Errorf("items.type = %v, want integer (unchanged)", items["type"])
	}
}

func TestSanitizeSchema_RemovePropertiesFromNonObject(t *testing.T) {
	schema := map[string]any{
		"type": "string",
		"properties": map[string]any{
			"x": map[string]any{"type": "string"},
		},
		"required": []any{"x"},
	}
	result := SanitizeSchema(schema)
	if _, ok := result["properties"]; ok {
		t.Error("properties should be removed from non-object type")
	}
	if _, ok := result["required"]; ok {
		t.Error("required should be removed from non-object type")
	}
}

func TestSanitizeSchema_NestedObjects(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"address": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"street": map[string]any{"type": "string"},
				},
				"additionalProperties": true,
			},
		},
	}
	result := SanitizeSchema(schema)
	props := result["properties"].(map[string]any)
	addr := props["address"].(map[string]any)
	if _, ok := addr["additionalProperties"]; ok {
		t.Error("nested additionalProperties should be removed")
	}
}

func TestSanitizeSchema_ArrayWithNestedSchema(t *testing.T) {
	schema := map[string]any{
		"type": "array",
		"items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id": map[string]any{"type": "integer"},
			},
			"additionalProperties": false,
		},
	}
	result := SanitizeSchema(schema)
	items := result["items"].(map[string]any)
	if _, ok := items["additionalProperties"]; ok {
		t.Error("nested additionalProperties should be removed")
	}
}

func TestSanitizeImpl_ScalarPassthrough(t *testing.T) {
	// Non-map, non-array, non-nil values pass through unchanged.
	result := sanitizeImpl("hello")
	if result != "hello" {
		t.Errorf("expected hello, got %v", result)
	}

	result = sanitizeImpl(42)
	if result != 42 {
		t.Errorf("expected 42, got %v", result)
	}

	result = sanitizeImpl(true)
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestSanitizeImpl_Array(t *testing.T) {
	input := []any{
		map[string]any{"type": "string", "additionalProperties": true},
		"plain",
	}
	result := sanitizeImpl(input).([]any)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	// First element (map) should have additionalProperties removed.
	m := result[0].(map[string]any)
	if _, ok := m["additionalProperties"]; ok {
		t.Error("additionalProperties should be removed from array element")
	}
	// Second element (string) passes through.
	if result[1] != "plain" {
		t.Errorf("result[1] = %v", result[1])
	}
}

func TestSanitizeImpl_Nil(t *testing.T) {
	result := sanitizeImpl(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSanitizeSchema_EnumNotArray(t *testing.T) {
	// enum is not an array -- should pass through as a regular value.
	schema := map[string]any{
		"type": "string",
		"enum": "not-an-array",
	}
	result := SanitizeSchema(schema)
	if result["enum"] != "not-an-array" {
		t.Errorf("enum = %v", result["enum"])
	}
}

func TestSanitizeSchema_RequiredNonStringEntries(t *testing.T) {
	// required array with non-string entries -- should be filtered out.
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name", 42},
	}
	result := SanitizeSchema(schema)
	required := result["required"].([]any)
	if len(required) != 1 {
		t.Fatalf("required = %v, want [name]", required)
	}
}

func TestSanitizeSchema_ObjectWithoutProperties(t *testing.T) {
	// object type without properties -- required filtering skipped.
	schema := map[string]any{
		"type":     "object",
		"required": []any{"name"},
	}
	result := SanitizeSchema(schema)
	// Without properties map, the required filtering branch (line 71 props check) is skipped.
	// required stays as-is.
	if result["type"] != "object" {
		t.Errorf("type = %v", result["type"])
	}
}

func TestSanitizeSchema_EnumWithNoType(t *testing.T) {
	// enum present but no type field.
	schema := map[string]any{
		"enum": []any{1, 2},
	}
	result := SanitizeSchema(schema)
	enumArr := result["enum"].([]any)
	if enumArr[0] != "1" {
		t.Errorf("enum[0] = %v, want '1'", enumArr[0])
	}
}

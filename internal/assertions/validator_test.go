package assertions

import (
	"testing"
	"time"

	"github.com/pc0stas/tango/internal/config"
	httppkg "github.com/pc0stas/tango/internal/http"
)

func strPtr(s string) *string { return &s }

func TestValidate_Status(t *testing.T) {
	tests := []struct {
		name     string
		expected int
		actual   int
		wantErr  bool
	}{
		{"match", 200, 200, false},
		{"mismatch", 200, 404, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(&config.ExpectSpec{Status: tt.expected}, &httppkg.Response{StatusCode: tt.actual})
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_ResponseTime(t *testing.T) {
	limit := 100
	err := Validate(
		&config.ExpectSpec{Status: 200, ResponseTime: &limit},
		&httppkg.Response{StatusCode: 200, Duration: 200 * time.Millisecond},
	)
	if err == nil {
		t.Error("expected error for exceeded response time")
	}

	err = Validate(
		&config.ExpectSpec{Status: 200, ResponseTime: &limit},
		&httppkg.Response{StatusCode: 200, Duration: 50 * time.Millisecond},
	)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateAssertion_Equals(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "name", Equals: "John"}, `{"name":"John"}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "name", Equals: "John"}, `{"name":"Jane"}`)
	if err == nil {
		t.Error("expected error for value mismatch")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "count", Equals: float64(3)}, `{"count":3}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateAssertion_Type(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		path     string
		typ      string
		wantErr  bool
	}{
		{"number", `{"v":42}`, "v", "number", false},
		{"string", `{"v":"hi"}`, "v", "string", false},
		{"boolean", `{"v":true}`, "v", "boolean", false},
		{"array", `{"v":[1,2]}`, "v", "array", false},
		{"object", `{"v":{"a":1}}`, "v", "object", false},
		{"null", `{"v":null}`, "v", "null", false},
		{"mismatch", `{"v":"hi"}`, "v", "number", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssertion(&config.AssertionSpec{Path: tt.path, Type: tt.typ}, tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("Type assertion error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAssertion_Contains(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "msg", Contains: "success"}, `{"msg":"operation successful"}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "msg", Contains: "success"}, `{"msg":"failed"}`)
	if err == nil {
		t.Error("expected error for missing substring")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "count", Contains: "x"}, `{"count":42}`)
	if err == nil {
		t.Error("expected error for non-string value")
	}
}

func TestValidateAssertion_Startswith(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "code", Startswith: "OK"}, `{"code":"OK_DONE"}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "code", Startswith: "OK"}, `{"code":"ERR_DONE"}`)
	if err == nil {
		t.Error("expected error for wrong prefix")
	}
}

func TestValidateAssertion_Endswith(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "code", Endswith: "_DONE"}, `{"code":"OK_DONE"}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "code", Endswith: "_DONE"}, `{"code":"OK_FAIL"}`)
	if err == nil {
		t.Error("expected error for wrong suffix")
	}
}

func TestValidateAssertion_Matches(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "email", Matches: `^[\w.-]+@[\w.-]+\.\w+$`}, `{"email":"test@example.com"}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "email", Matches: `^[\w.-]+@[\w.-]+\.\w+$`}, `{"email":"invalid"}`)
	if err == nil {
		t.Error("expected error for non-matching regex")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "email", Matches: `[`}, `{"email":"test"}`)
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestValidateAssertion_GreaterThan(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "n", GreaterThan: float64(10)}, `{"n":15}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "n", GreaterThan: float64(10)}, `{"n":5}`)
	if err == nil {
		t.Error("expected error for smaller value")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "n", GreaterThan: float64(10)}, `{"n":10}`)
	if err == nil {
		t.Error("expected error for equal value (not strictly greater)")
	}
}

func TestValidateAssertion_LessThanOrEqual(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "n", LessThanOrEqual: float64(10)}, `{"n":5}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "n", LessThanOrEqual: float64(10)}, `{"n":10}`)
	if err != nil {
		t.Errorf("expected no error for equal value, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "n", LessThanOrEqual: float64(10)}, `{"n":15}`)
	if err == nil {
		t.Error("expected error for greater value")
	}
}

func TestValidateAssertion_Between(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "age", Between: []any{float64(18), float64(65)}}, `{"age":30}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "age", Between: []any{float64(18), float64(65)}}, `{"age":18}`)
	if err != nil {
		t.Errorf("expected no error at lower bound, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "age", Between: []any{float64(18), float64(65)}}, `{"age":65}`)
	if err != nil {
		t.Errorf("expected no error at upper bound, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "age", Between: []any{float64(18), float64(65)}}, `{"age":17}`)
	if err == nil {
		t.Error("expected error for value below range")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "age", Between: []any{float64(18)}}, `{"age":30}`)
	if err == nil {
		t.Error("expected error for wrong number of bounds")
	}
}

func TestValidateAssertion_Length(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "items", Length: 3}, `{"items":[1,2,3]}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "items", Length: 3}, `{"items":[1,2]}`)
	if err == nil {
		t.Error("expected error for wrong length")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "items", Length: 3}, `{"items":"str"}`)
	if err == nil {
		t.Error("expected error for non-array value")
	}
}

func TestValidateAssertion_MinLength(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "items", MinLength: 2}, `{"items":[1,2,3]}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "items", MinLength: 2}, `{"items":[1]}`)
	if err == nil {
		t.Error("expected error for below min_length")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "items", MinLength: 2}, `{"items":{}}`)
	if err == nil {
		t.Error("expected error for non-array value")
	}
}

func TestValidateAssertion_Empty(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name    string
		spec    config.AssertionSpec
		json    string
		wantErr bool
	}{
		{"empty string true", config.AssertionSpec{Path: "v", Empty: &trueVal}, `{"v":""}`, false},
		{"non-empty string true", config.AssertionSpec{Path: "v", Empty: &trueVal}, `{"v":"hi"}`, true},
		{"empty array true", config.AssertionSpec{Path: "v", Empty: &trueVal}, `{"v":[]}`, false},
		{"non-empty array true", config.AssertionSpec{Path: "v", Empty: &trueVal}, `{"v":[1]}`, true},
		{"null true", config.AssertionSpec{Path: "v", Empty: &trueVal}, `{"v":null}`, false},
		{"non-empty string false", config.AssertionSpec{Path: "v", Empty: &falseVal}, `{"v":"hi"}`, false},
		{"empty string false", config.AssertionSpec{Path: "v", Empty: &falseVal}, `{"v":""}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssertion(&tt.spec, tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("Empty assertion error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAssertion_Exists(t *testing.T) {
	trueVal := true
	falseVal := false

	tests := []struct {
		name    string
		spec    config.AssertionSpec
		json    string
		wantErr bool
	}{
		{"exists true and path exists", config.AssertionSpec{Path: "v", Exists: &trueVal}, `{"v":42}`, false},
		{"exists true but path missing", config.AssertionSpec{Path: "x", Exists: &trueVal}, `{"v":42}`, true},
		{"exists false and path missing", config.AssertionSpec{Path: "x", Exists: &falseVal}, `{"v":42}`, false},
		{"exists false but path exists", config.AssertionSpec{Path: "v", Exists: &falseVal}, `{"v":42}`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAssertion(&tt.spec, tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("Exists assertion error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAssertion_ContainsValue(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "roles", ContainsValue: "admin"}, `{"roles":["user","admin","owner"]}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "roles", ContainsValue: "admin"}, `{"roles":["user","owner"]}`)
	if err == nil {
		t.Error("expected error for missing value")
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "roles", ContainsValue: "admin"}, `{"roles":"not_array"}`)
	if err == nil {
		t.Error("expected error for non-array value")
	}
}

func TestValidateAssertion_DeepEquals(t *testing.T) {
	expected := map[string]any{
		"created_by": "system",
		"version":    float64(1),
	}

	err := ValidateAssertion(&config.AssertionSpec{Path: "meta", DeepEquals: expected}, `{"meta":{"created_by":"system","version":1}}`)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	diff := map[string]any{
		"created_by": "user",
		"version":    float64(2),
	}

	err = ValidateAssertion(&config.AssertionSpec{Path: "meta", DeepEquals: diff}, `{"meta":{"created_by":"system","version":1}}`)
	if err == nil {
		t.Error("expected error for deep equality mismatch")
	}
}

func TestValidateAssertion_PathNotFound(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "nonexistent", Equals: "x"}, `{"v":42}`)
	if err == nil {
		t.Error("expected error for missing path")
	}
}

func TestValidateAssertion_NoMatcher(t *testing.T) {
	err := ValidateAssertion(&config.AssertionSpec{Path: "v"}, `{"v":42}`)
	if err != nil {
		t.Errorf("expected no error when no matcher is set, got %v", err)
	}
}

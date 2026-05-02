package assertions

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/pc0stas/tango/internal/config"
	httppkg "github.com/pc0stas/tango/internal/http"
	"github.com/tidwall/gjson"
)

func Validate(expect *config.ExpectSpec, response *httppkg.Response) error {
	if response.StatusCode != expect.Status {
		return fmt.Errorf("status mismatch: expected %d, got %d", expect.Status, response.StatusCode)
	}

	for i, assertion := range expect.Body {
		if err := ValidateAssertion(&assertion, response.Body); err != nil {
			return fmt.Errorf("assertion %d failed: %w", i+1, err)
		}
	}

	if expect.ResponseTime != nil {
		if response.Duration.Milliseconds() > int64(*expect.ResponseTime) {
			return fmt.Errorf("response time exceeded: %dms > %dms",
				response.Duration.Milliseconds(), *expect.ResponseTime)
		}
	}

	return nil
}

func ValidateAssertion(assertion *config.AssertionSpec, body string) error {
	result := gjson.Get(body, assertion.Path)

	if !result.Exists() {
		if assertion.Exists != nil && !*assertion.Exists {
			return nil
		}
		return fmt.Errorf("path not found: %s", assertion.Path)
	}

	if assertion.Exists != nil {
		if !*assertion.Exists {
			return fmt.Errorf("path should not exist: %s", assertion.Path)
		}
		return nil
	}

	value := result.Value()

	switch {
	case assertion.Equals != nil:
		if !reflect.DeepEqual(value, assertion.Equals) {
			return fmt.Errorf("value mismatch: expected %v, got %v", assertion.Equals, value)
		}

	case assertion.Type != "":
		actual := getType(value)
		if actual != assertion.Type {
			return fmt.Errorf("type mismatch: expected %s, got %s", assertion.Type, actual)
		}

	case assertion.Contains != "":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}
		if !strings.Contains(str, assertion.Contains) {
			return fmt.Errorf("value does not contain %q", assertion.Contains)
		}

	case assertion.Startswith != "":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}
		if !strings.HasPrefix(str, assertion.Startswith) {
			return fmt.Errorf("value does not start with %q", assertion.Startswith)
		}

	case assertion.Endswith != "":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}
		if !strings.HasSuffix(str, assertion.Endswith) {
			return fmt.Errorf("value does not end with %q", assertion.Endswith)
		}

	case assertion.Matches != "":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("value is not a string")
		}
		matched, err := regexp.MatchString(assertion.Matches, str)
		if err != nil {
			return fmt.Errorf("invalid regex: %w", err)
		}
		if !matched {
			return fmt.Errorf("value does not match pattern %q", assertion.Matches)
		}

	case assertion.GreaterThan != nil:
		got, ok := toFloat64(value)
		if !ok {
			return fmt.Errorf("value is not a number")
		}
		expected, ok := toFloat64(assertion.GreaterThan)
		if !ok {
			return fmt.Errorf("greater_than is not a number")
		}
		if !(got > expected) {
			return fmt.Errorf("value %v is not greater than %v", got, expected)
		}

	case assertion.LessThanOrEqual != nil:
		got, ok := toFloat64(value)
		if !ok {
			return fmt.Errorf("value is not a number")
		}
		expected, ok := toFloat64(assertion.LessThanOrEqual)
		if !ok {
			return fmt.Errorf("less_than_or_equal is not a number")
		}
		if !(got <= expected) {
			return fmt.Errorf("value %v is not <= %v", got, expected)
		}

	case assertion.Between != nil:
		if len(assertion.Between) != 2 {
			return fmt.Errorf("between requires exactly 2 values")
		}
		got, ok := toFloat64(value)
		if !ok {
			return fmt.Errorf("value is not a number")
		}
		lo, ok := toFloat64(assertion.Between[0])
		if !ok {
			return fmt.Errorf("between lower bound is not a number")
		}
		hi, ok := toFloat64(assertion.Between[1])
		if !ok {
			return fmt.Errorf("between upper bound is not a number")
		}
		if got < lo || got > hi {
			return fmt.Errorf("value %v is not between %v and %v", got, lo, hi)
		}

	case assertion.Length > 0:
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("value is not an array")
		}
		if len(arr) != assertion.Length {
			return fmt.Errorf("length mismatch: expected %d, got %d", assertion.Length, len(arr))
		}

	case assertion.MinLength > 0:
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("value is not an array")
		}
		if len(arr) < assertion.MinLength {
			return fmt.Errorf("min_length: expected >= %d, got %d", assertion.MinLength, len(arr))
		}

	case assertion.Empty != nil:
		isEmpty := checkEmpty(value)
		if isEmpty != *assertion.Empty {
			return fmt.Errorf("empty check failed: expected %v, got %v", *assertion.Empty, isEmpty)
		}

	case assertion.ContainsValue != nil:
		arr, ok := value.([]any)
		if !ok {
			return fmt.Errorf("value is not an array")
		}
		found := false
		for _, item := range arr {
			if reflect.DeepEqual(item, assertion.ContainsValue) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("array does not contain %v", assertion.ContainsValue)
		}

	case assertion.DeepEquals != nil:
		if !reflect.DeepEqual(value, assertion.DeepEquals) {
			return fmt.Errorf("deep equality mismatch: expected %v, got %v", assertion.DeepEquals, value)
		}
	}

	return nil
}

func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func getType(v any) string {
	switch v.(type) {
	case float64:
		return "number"
	case string:
		return "string"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func checkEmpty(v any) bool {
	switch val := v.(type) {
	case string:
		return val == ""
	case []any:
		return len(val) == 0
	case map[string]any:
		return len(val) == 0
	case nil:
		return true
	default:
		return false
	}
}

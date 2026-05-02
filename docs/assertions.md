# Assertions

Assertions validate the HTTP response. They go under `expect.body` as a list.

## All matchers

Only **one matcher** is active per assertion block — the first one set wins.

### Equality

```yaml
- path: "name"
  equals: "Pablo"            # Exact value match
```

Compares using `reflect.DeepEqual`. Handles numeric coercion: YAML `int(1)` and JSON `float64(1)` are considered equal.

### Type check

```yaml
- path: "id"
  type: "number"             # number, string, boolean, array, object, null
```

### String operations

```yaml
- path: "message"
  contains: "success"        # Substring check
- path: "status"
  startswith: "OK"           # Prefix check
- path: "code"
  endswith: "_DONE"          # Suffix check
```

### Regex

```yaml
- path: "email"
  matches: "^[\\w.-]+@[\\w.-]+\\.\\w+$"
```

### Numeric comparisons

```yaml
- path: "count"
  greater_than: 0            # Strict > comparison
- path: "score"
  less_than_or_equal: 100    # <= comparison
- path: "age"
  between: [18, 65]          # Inclusive range
```

Values are coerced to `float64` for comparison. Strings that parse as numbers are accepted.

### Array checks

```yaml
- path: "items"
  length: 5                  # Exact length
- path: "items"
  min_length: 1              # Minimum length >= N
- path: "roles"
  contains_value: "admin"    # Array contains specific value
```

### Emptiness

```yaml
- path: "field"
  empty: true                # true = must be empty, false = must not be empty
```

Empty means: empty string, empty array, empty object, or null.

### Existence

```yaml
- path: "optional_field"
  exists: true               # true = path must exist, false = path must not exist
```

When `exists: false` and the path doesn't exist, the assertion passes. When the path exists but `exists: false`, it fails.

### Deep equality

```yaml
- path: "metadata"
  deep_equals:
    key1: value1
    key2: value2
```

Compares the entire JSON subtree using `reflect.DeepEqual`.

## Header assertions

```yaml
expect:
  headers:
    Content-Type: "application/json"
```

Headers support simple string matching only.

## Response time

```yaml
expect:
  response_time_ms: 5000     # Fail if response took longer than 5s
```

# Retry Strategy

Retry failed requests with exponential backoff.

## Configuration

```yaml
retry:
  count: 3                     # Max retry attempts (default: 0 = no retries)
  delay: 1s                    # Initial delay (default: 1s)
  backoff_multiplier: 2.0      # Exponential backoff (default: 2.0)
  retry_on_status: [502, 503]  # Status codes that trigger retry
```

## Behavior

- If `count` is 0, the request executes once with no retries
- Each retry sleeps before the next attempt
- The delay grows exponentially: `delay * multiplier^(attempt-1)`
  - Attempt 1: first request (no delay)
  - Attempt 2: `delay` (1s)
  - Attempt 3: `delay * multiplier` (2s)
  - Attempt 4: `delay * multiplier^2` (4s)
- Retries stop when the response status is **not** in `retry_on_status`, or max attempts reached
- Connection errors break the retry loop immediately

## Example

```yaml
- name: "flaky_endpoint"
  request:
    url: "https://api.example.com/unstable"
  retry:
    count: 5
    delay: 500ms
    backoff_multiplier: 2.0
    retry_on_status: [408, 429, 500, 502, 503, 504]
  expect:
    status: 200
```

With this configuration, if the endpoint returns 503:
1. Try → 503 → sleep 500ms
2. Try → 503 → sleep 1s
3. Try → 503 → sleep 2s
4. Try → 503 → sleep 4s
5. Try → 503 → sleep 8s
6. Try → (last attempt, stop regardless)

## Default values

When `retry` is not specified:

| Field | Default |
|---|---|
| `count` | 0 |
| `delay` | 1s |
| `backoff_multiplier` | 2.0 |

No retries happen by default.

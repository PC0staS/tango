# Tango — Event Streaming & Distributed Testing CLI

**Language:** Go  
**Purpose:** Local event bus + CLI testing framework for coordinated workflow validation

---

## Part 1: Event Streaming (Phase 2)
*Skipped for now; focus on Testing first*

---

## Part 2: Testing CLI — MVP Spec

### Overview
Tango reads a YAML workflow file, executes HTTP requests in dependency order, validates responses, and reports results with timing and status.

---

## 1. Core YAML Schema

### Top-level Structure
```yaml
name: "my-workflow-test"
description: "Test my Pi services end-to-end"
version: "1.0"
config:
  timeout_default: 10s
  retry_failed_steps: false
  stop_on_error: true
  parallel_safe: false  # Execute steps sequentially

steps:
  - # Step definition (see below)
```

### Step Definition (Full)
```yaml
- name: "step_identifier"
  description: "Optional human-readable description"
  
  # Request config
  request:
    method: "POST"                    # GET, POST, PUT, DELETE, PATCH
    url: "http://localhost:8000/api"
    headers:
      Content-Type: "application/json"
      Authorization: "Bearer token123"
    body: |                           # Raw string or JSON
      {
        "key": "value",
        "nested": {
          "data": 42
        }
      }
    body_file: "/path/to/body.json"   # Alternative: read from file
    query_params:
      filter: "active"
      limit: "10"
    
  # Validation
  expect:
    status: 200                        # HTTP status code (required)
    headers:
      Content-Type: "application/json"
      X-Custom-Header: "expected-value"
    body:
      - path: "$.status"               # JSONPath
        equals: "success"
      - path: "$.data[0].id"
        type: "number"
      - path: "$.message"
        contains: "created"
      - path: "$.timestamp"
        matches: "^\\d{4}-\\d{2}.*"    # Regex
      - path: "$.errors"
        empty: true
    response_time_ms: 1000             # Max response time (optional)
  
  # Dependencies & Control Flow
  depends_on:
    - "previous_step_name"             # List of steps that must finish first
    - "another_step"
  
  timeout: 5s                          # Override default timeout
  skip: false                          # Skip this step
  skip_if_failed: "parent_step"        # Skip if another step fails
  
  # Conditional Execution
  run_if:
    previous_step: "success"           # success, failed, skipped
    env_var: "ENV_VAR_NAME=value"
  
  # Data Capture & Reuse
  capture:
    response_body: "response"          # Store full response
    json_path: 
      user_id: "$.data.id"             # Extract specific fields
      auth_token: "$.token"
      created_at: "$.timestamp"
  
  # Retries
  retry:
    count: 3
    delay: 1s
    backoff_multiplier: 2.0            # Exponential backoff
    retry_on_status: [408, 429, 500, 502, 503]
  
  # Post-request actions
  extract_and_store: true              # Store captured data for later steps
  log_response: true                   # Log full response body
```

---

## 2. JSONPath & Assertions

### Supported Assertion Types

```yaml
expect:
  body:
    # Direct equality
    - path: "$.user.name"
      equals: "Pablo"
    
    # Type checking
    - path: "$.id"
      type: "number"  # number, string, boolean, array, object, null
    
    # String operations
    - path: "$.message"
      contains: "success"
    - path: "$.status"
      startswith: "OK"
    - path: "$.code"
      endswith: "_DONE"
    
    # Regex matching
    - path: "$.email"
      matches: "^[\\w.-]+@[\\w.-]+\\.\\w+$"
    
    # Numeric comparisons
    - path: "$.count"
      greater_than: 0
    - path: "$.score"
      less_than_or_equal: 100
    - path: "$.age"
      between: [18, 65]
    
    # Array/Collection checks
    - path: "$.items"
      length: 5
    - path: "$.items"
      min_length: 1
    - path: "$.items"
      empty: false
    
    # Existence checks
    - path: "$.optional_field"
      exists: true
    - path: "$.removed_field"
      exists: false
    
    # Array element checks
    - path: "$.roles"
      contains_value: "admin"
    
    # Deep equality (full object)
    - path: "$.metadata"
      deep_equals:
        created_by: "system"
        version: 1
```

### Header Assertions
```yaml
expect:
  headers:
    Content-Type: "application/json"
    X-Request-ID: 
      matches: "^[a-f0-9-]{36}$"
    X-Rate-Limit-Remaining:
      greater_than: 0
```

---

## 3. Variable Substitution & Capture

### Captured Variables (From Previous Steps)
```yaml
steps:
  - name: "create_user"
    request:
      method: "POST"
      url: "http://localhost:8000/users"
      body: |
        {
          "name": "Pablo",
          "email": "pablo@example.com"
        }
    capture:
      user_id: "$.id"
      auth_token: "$.token"
  
  - name: "update_user"
    depends_on: ["create_user"]
    request:
      method: "PUT"
      url: "http://localhost:8000/users/{{ steps.create_user.user_id }}"
      headers:
        Authorization: "Bearer {{ steps.create_user.auth_token }}"
      body: |
        {
          "name": "Pablo Updated"
        }
```

### Environment Variables
```yaml
steps:
  - name: "fetch_data"
    request:
      method: "GET"
      url: "{{ env.API_BASE_URL }}/data"
      headers:
        Authorization: "Bearer {{ env.API_KEY }}"
```

### Config Variables
```yaml
config:
  variables:
    base_url: "http://localhost:8000"
    timeout_default: 10s

steps:
  - name: "health_check"
    request:
      method: "GET"
      url: "{{ config.base_url }}/health"
      timeout: "{{ config.timeout_default }}"
```

---

## 4. Advanced Flow Control

### Conditional Execution
```yaml
- name: "skip_if_env"
  skip_if:
    - env: "SKIP_TESTS"
      equals: "true"
    - env: "ENVIRONMENT"
      equals: "production"

- name: "run_only_on_success"
  run_if:
    previous_step: "create_user"
    status: "success"

- name: "run_only_if_var_set"
  run_if:
    env_var_exists: "SPECIAL_MODE"
```

### Retry Strategy
```yaml
- name: "unstable_endpoint"
  request:
    method: "GET"
    url: "http://flaky-service:9000/data"
  
  retry:
    count: 5
    initial_delay: 500ms
    backoff_multiplier: 2.0
    max_delay: 30s
    retry_on_status: [408, 429, 500, 502, 503, 504]
    retry_on_error: true  # Retry on connection errors
```

### Loop/Iteration
```yaml
- name: "bulk_create"
  loop:
    items: ["item1", "item2", "item3"]
    variable: "current_item"
  request:
    method: "POST"
    url: "http://localhost:8000/items"
    body: |
      {
        "name": "{{ loop.current_item }}"
      }
  capture:
    item_ids: "$.id"  # Collects all IDs from iterations
```

---

## 5. Output & Reporting

### CLI Output Format
```
$ tango test workflow.yaml

[✓] create_user (POST http://localhost:8000/users) — 201 Created [245ms]
  └─ Captured: user_id=42, auth_token=abc123xyz

[✓] update_user (PUT http://localhost:8000/users/42) — 200 OK [128ms]

[✗] delete_user (DELETE http://localhost:8000/users/999) — 404 Not Found [89ms]
  └─ Assertion failed: expect status 204, got 404
  └─ Response: {"error": "User not found"}

[⊘] cleanup (POST ...) — SKIPPED (depends on delete_user which failed)

─────────────────────────────────────────
Summary: 2 passed, 1 failed, 1 skipped (out of 4 steps)
Total time: 462ms
Exit code: 1
```

### JSON Report Output
```bash
$ tango test workflow.yaml --output=json > report.json
```

```json
{
  "name": "my-workflow-test",
  "timestamp": "2026-04-27T10:30:45Z",
  "total_steps": 4,
  "passed": 2,
  "failed": 1,
  "skipped": 1,
  "total_duration_ms": 462,
  "steps": [
    {
      "name": "create_user",
      "status": "passed",
      "method": "POST",
      "url": "http://localhost:8000/users",
      "request_duration_ms": 245,
      "response_status": 201,
      "response_headers": { "Content-Type": "application/json" },
      "response_body": "{...}",
      "captured_vars": {
        "user_id": "42",
        "auth_token": "abc123xyz"
      }
    }
  ]
}
```

---

## 6. CLI Commands & Flags

### Main Commands
```bash
# Run a test workflow
tango test <workflow.yaml> [flags]

# Validate workflow syntax
tango validate <workflow.yaml>

# Generate a template
tango init <name>

# Show help
tango help
```

### Flags
```bash
tango test workflow.yaml \
  --env-file=.env.test           # Load env vars from file
  --output=json                  # Output format: text (default), json, junit
  --verbose                      # Verbose logging
  --stop-on-error                # Exit on first failure
  --parallel                     # Run steps in parallel (if safe)
  --step=create_user             # Run only specific step(s)
  --exclude-step=cleanup         # Exclude step(s)
  --timeout=30s                  # Global timeout override
  --var="KEY=value"              # Set variables from CLI
  --report=report.json           # Save JSON report
  --junit=report.xml             # Save JUnit XML (for CI)
```

### Examples
```bash
# Run full workflow
tango test pi_services.yaml

# Run only auth steps, verbose
tango test pi_services.yaml --step=login --step=verify --verbose

# Run with custom env
tango test pi_services.yaml --env-file=.env.prod --stop-on-error

# Generate report for CI
tango test pi_services.yaml --junit=test-results.xml
```

---

## 7. File Structure & Best Practices

### Suggested Project Layout
```
my-tests/
├── workflows/
│   ├── auth.yaml
│   ├── pi_services.yaml
│   ├── esp32_integration.yaml
│   └── toolbox_api.yaml
├── fixtures/
│   ├── user_payload.json
│   ├── bulk_items.json
│   └── error_response.json
├── .env.test
├── .env.prod
├── tango.config.yaml           # Optional global config
└── README.md
```

### Reusable Workflow Composition
```yaml
# workflows/auth.yaml
name: "authentication"
steps:
  - name: "login"
    request:
      method: "POST"
      url: "{{ config.base_url }}/auth/login"
      body_file: "fixtures/login_payload.json"
    capture:
      auth_token: "$.token"

# workflows/full_integration.yaml
imports:
  - "workflows/auth.yaml"

config:
  base_url: "http://localhost:8000"

steps:
  - name: "use_auth_token"
    depends_on: ["login"]  # From imported workflow
    request:
      method: "GET"
      url: "{{ config.base_url }}/protected"
      headers:
        Authorization: "Bearer {{ steps.login.auth_token }}"
```

---

## 8. Error Handling & Debugging

### Assertion Failure Details
```
[✗] Step: update_user
    Request: PUT http://localhost:8000/users/42
    Status: Expected 204, got 400
    
    Assertion 1 (PASS): $.id exists
    Assertion 2 (FAIL): $.message contains "updated"
      Expected substring: "updated"
      Actual value: "Validation error: invalid email format"
    
    Full Response Body:
    {
      "error": true,
      "message": "Validation error: invalid email format",
      "field": "email"
    }
    
    Captured Variables:
    None
```

### Verbose Mode Output
```bash
tango test workflow.yaml --verbose

[DEBUG] Parsing workflow.yaml
[DEBUG] Found 4 steps
[DEBUG] Loading .env variables (5 vars)
[DEBUG] Step 'create_user' — No dependencies, executing
[DEBUG] Resolving URL: {{ config.base_url }}/users → http://localhost:8000/users
[DEBUG] Resolving headers...
[DEBUG] Request sent: POST http://localhost:8000/users [body: 156 bytes]
[DEBUG] Response received: 201 [body: 342 bytes, time: 245ms]
[DEBUG] Validating status: expect 201, got 201 ✓
[DEBUG] Validating body assertions (3 assertions)
[DEBUG]   → $.id exists: ✓
[DEBUG]   → $.id type number: ✓
[DEBUG]   → $.token matches regex: ✓
[DEBUG] Capturing variables: user_id=42, auth_token=xyz
[DEBUG] Step 'update_user' — Dependency 'create_user' ready, executing
...
```

---

## 9. Integration Points

### GitHub Actions CI
```yaml
# .github/workflows/test.yml
name: Tango Tests
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - run: go install github.com/pc0stas/tango@latest
      - run: tango test workflows/full_integration.yaml --junit=results.xml
      - uses: dorny/test-reporter@v1
        if: always()
        with:
          name: Tango Test Results
          path: results.xml
          reporter: java-junit
```

### Docker
```yaml
# docker-compose.yml (for test environment)
version: '3'
services:
  app:
    image: my-app:latest
    ports:
      - "8000:8000"
  
  tango:
    image: golang:1.23
    volumes:
      - .:/workspace
    working_dir: /workspace
    command: tango test workflows/integration.yaml
    depends_on:
      - app
```

---

## 10. MVP Roadmap

### Phase 1: Core (Week 1-2)
- ✓ YAML parsing (top-level, steps, basic request)
- ✓ HTTP client (GET, POST, PUT, DELETE)
- ✓ Status code validation
- ✓ Dependency resolution & topological sort
- ✓ Sequential execution
- ✓ CLI output (text format)
- ✓ Variable substitution (env, config)
- ✓ JSONPath assertions (equals, contains, exists)
- ✓ Capture variables from responses

### Phase 2: Enhanced Assertions & Control (Week 3)
- ✓ Advanced assertion types (regex, numeric, array)
- ✓ Header assertions & validation
- ✓ Retry logic with backoff
- ✓ Conditional execution (run_if, skip_if)
- ✓ Timeout per step
- ✓ JSON report output

### Phase 3: Advanced Features (Week 4+)
- ✓ Loops/iteration
- ✓ Workflow composition (imports)
- ✓ JUnit XML output (for CI)
- ✓ Parallel execution (safe mode)
- ✓ Request body from files
- ✓ Verbose/debug logging

### Phase 4: Ecosystem (Future)
- Dashboard / Web UI
- Integration with n8n
- Event streaming (Part 1)
- Distributed tracing

---

## 11. Example Workflows

### Simple Health Check
```yaml
name: "health-check"
version: "1.0"

config:
  timeout_default: 5s

steps:
  - name: "pi-health"
    request:
      method: "GET"
      url: "http://192.168.1.102:8080/health"
    expect:
      status: 200
      body:
        - path: "$.status"
          equals: "ok"
```

### Pi Services Full Integration Test
```yaml
name: "pi-full-integration"
description: "Test all Pi homelab services"
version: "1.0"

config:
  base_url: "http://192.168.1.102"
  timeout_default: 10s
  variables:
    test_file: "test.txt"

steps:
  - name: "check-pihole"
    request:
      method: "GET"
      url: "{{ config.base_url }}:80/admin/api.php?status"
    expect:
      status: 200
  
  - name: "check-nextcloud"
    request:
      method: "GET"
      url: "{{ config.base_url }}/nextcloud"
    expect:
      status: 200
      response_time_ms: 2000
  
  - name: "check-grafana"
    request:
      method: "GET"
      url: "{{ config.base_url }}:3000/api/health"
    expect:
      status: 200
      body:
        - path: "$.database"
          equals: "ok"
  
  - name: "mosquitto-publish"
    request:
      method: "POST"
      url: "{{ config.base_url }}:1883/publish"
      body: |
        {
          "topic": "test/tango",
          "message": "test"
        }
    expect:
      status: 200
  
  - name: "esp32-sensor-read"
    depends_on: ["mosquitto-publish"]
    request:
      method: "GET"
      url: "{{ config.base_url }}:5000/sensors"
    expect:
      status: 200
      body:
        - path: "$.sensors"
          min_length: 1
    capture:
      sensor_data: "$.sensors[0]"
  
  - name: "n8n-trigger"
    request:
      method: "POST"
      url: "{{ config.base_url }}:5678/webhook/test"
      body: |
        {
          "trigger": "tango-test"
        }
    expect:
      status: 200
```

### Toolbox API with Auth
```yaml
name: "toolbox-api-auth"
version: "1.0"

config:
  base_url: "{{ env.TOOLBOX_API_URL }}"

steps:
  - name: "register"
    request:
      method: "POST"
      url: "{{ config.base_url }}/auth/register"
      body: |
        {
          "email": "test@example.com",
          "password": "SecurePass123!"
        }
    expect:
      status: 201
      body:
        - path: "$.user.id"
          type: "number"
        - path: "$.token"
          type: "string"
    capture:
      user_id: "$.user.id"
      auth_token: "$.token"
  
  - name: "login"
    request:
      method: "POST"
      url: "{{ config.base_url }}/auth/login"
      body: |
        {
          "email": "test@example.com",
          "password": "SecurePass123!"
        }
    expect:
      status: 200
      body:
        - path: "$.token"
          exists: true
    capture:
      new_token: "$.token"
  
  - name: "upload-image"
    depends_on: ["login"]
    request:
      method: "POST"
      url: "{{ config.base_url }}/compress"
      headers:
        Authorization: "Bearer {{ steps.login.new_token }}"
      body_file: "fixtures/test_image.jpg"
    expect:
      status: 200
      response_time_ms: 5000
  
  - name: "get-user-profile"
    depends_on: ["login"]
    request:
      method: "GET"
      url: "{{ config.base_url }}/users/{{ steps.register.user_id }}"
      headers:
        Authorization: "Bearer {{ steps.login.new_token }}"
    expect:
      status: 200
      body:
        - path: "$.email"
          equals: "test@example.com"
  
  - name: "cleanup"
    depends_on: ["get-user-profile"]
    request:
      method: "DELETE"
      url: "{{ config.base_url }}/users/{{ steps.register.user_id }}"
      headers:
        Authorization: "Bearer {{ steps.login.new_token }}"
    expect:
      status: 204
```

---

## 12. Implementation Notes

### Parsing Strategy
- Use `gopkg.in/yaml.v3` for YAML
- Struct tags for unmarshaling
- Post-processing to resolve dependencies

### Execution Engine
- Build dependency DAG using topological sort
- Sequential execution (MVP) with channel-based coordination
- Variable substitution with `text/template` or regex
- JSONPath extraction via `tidwall/gjson` or similar

### HTTP Client
- Use `net/http` with custom transport for retries
- Timeout per request
- Response capture (headers, body)

### Testing & Validation
- Unit tests for YAML parsing
- Integration tests with mock HTTP server
- CLI tests with `testing/quick` or similar

---

## 13. Success Criteria (MVP Done When)

1. ✓ Can parse valid YAML workflows
2. ✓ Executes HTTP requests in dependency order
3. ✓ Validates status codes and basic body assertions
4. ✓ Captures and reuses variables between steps
5. ✓ Retries with backoff
6. ✓ Produces clear CLI output + JSON report
7. ✓ Handles env vars and config substitution
8. ✓ Works with 2-3 real integration test cases (Pi, Toolbox, etc)
9. ✓ Documented with examples
10. ✓ Publishable to GitHub Releases / Homebrew

---

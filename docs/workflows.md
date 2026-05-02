# Workflow Reference

A workflow is a YAML file that defines a sequence of HTTP requests to execute.

## Structure

```yaml
name: "workflow-name"          # Required. Unique name
description: "..."             # Optional description

config:                        # Global settings
  timeout_default: 10s         # Default timeout for all steps
  stop_on_error: false         # Stop execution on first failure
  variables:                   # Global variables (used with {{ .name }})
    host: "api.example.com"

steps:                         # List of steps to execute
  - name: "step-name"
    description: "..."
    request:                   # HTTP request definition
      method: "GET"            # GET, POST, PUT, DELETE, PATCH...
      url: "https://..."
      headers:                 # Request headers
        Content-Type: "application/json"
        Authorization: "Bearer token"
      body: |                  # Request body (raw string)
        {"key": "value"}
      body_file: "data.json"   # Load body from file
      query_params:            # URL query parameters
        page: "1"
    expect:                    # Response validation
      status: 200
      headers:
        Content-Type: "application/json"
      body:                    # JSONPath assertions
        - path: "name"
          equals: "Pablo"
      response_time_ms: 5000   # Max response time in ms
    depends_on:                # Wait for these steps to complete
      - "previous-step"
    timeout: 5s                # Override global timeout
    skip: true                 # Skip this step
    skip_if_failed: "step"     # Skip if another step failed
    run_if:                    # Conditionally execute
      previous_step: "step"
      status: "passed"
    capture:                   # Capture data from response
      json_path:
        var_name: "path"       # Extract JSONPath values
      response_body: "name"    # Store full response body
    retry:                     # Retry configuration
      count: 3
      delay: 1s
      backoff_multiplier: 2.0
      retry_on_status: [500, 502, 503]
```

## Execution order

Steps execute in dependency order (topological sort). Steps with `depends_on` wait for their dependencies. Implicit dependencies are derived automatically from:

- `run_if.previous_step` adds a dependency on that step
- `skip_if_failed: "step"` adds a dependency on that step
- `{{ steps.step.var }}` in URL, body, or headers adds a dependency on that step

No need to duplicate `depends_on` when these fields already express the dependency.

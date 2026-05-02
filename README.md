# Tango

Distributed API testing CLI with declarative YAML workflows.

## Install

```bash
go build -o tango .
```

## Quick start

```bash
./tango test examples/health_check.yaml
./tango validate examples/health_check.yaml
```

## Commands

| Command | Description |
|---------|-------------|
| `tango test <file.yaml>` | Execute a workflow |
| `tango validate <file.yaml>` | Validate YAML syntax and structure |

## Workflow structure

```yaml
name: "my-workflow"
description: "Example workflow"

config:
  timeout_default: 10s
  stop_on_error: false
  variables:
    host: "api.example.com"

steps:
  - name: "login"
    request:
      method: "POST"
      url: "https://{{ .host }}/login"
      headers:
        Content-Type: "application/json"
      body: '{"user":"admin","pass":"1234"}'
    expect:
      status: 200
      response_time_ms: 5000
    capture:
      json_path:
        token: "$.access_token"
    retry:
      count: 3
      delay: 1s
      backoff_multiplier: 2.0
      retry_on_status: [500, 502, 503]
```

## Variable substitution

Templates `{{ }}` are resolved before each request.

| Syntax | Source |
|--------|--------|
| `{{ .NAME }}` | Config variable (shorthand) |
| `{{ config.NAME }}` | Config variable |
| `{{ steps.STEP.VAR }}` | Captured from a previous step |
| `{{ env.NAME }}` | OS environment variable |

## Conditional execution

Control which steps run based on previous outcomes.

| Field | Description |
|-------|-------------|
| `skip: true` | Always skip this step |
| `skip_if_failed: "step_name"` | Skip if that step failed |
| `run_if.previous_step` | Only run if a specific step meets a condition |
| `run_if.status` | Required status: `"passed"`, `"failed"`, `"skipped"` |

```yaml
- name: "upload"
  run_if:
    previous_step: "create"
    status: "passed"
  request:
    method: "POST"
    url: "https://api.example.com/upload"

- name: "rollback"
  skip_if_failed: "deploy"
  request:
    method: "POST"
    url: "https://api.example.com/rollback"
```

## Retry strategy

```yaml
retry:
  count: 3              # Max retry attempts
  delay: 1s              # Initial delay
  backoff_multiplier: 2.0 # Exponential backoff
  retry_on_status: [500, 502, 503, 504]
```

Default retry (when not set): `count: 0`, `delay: 1s`, `backoff_multiplier: 2.0`.

## Assertions — all matchers

| Matcher | Example |
|---------|---------|
| `equals` | `equals: "Pablo"` |
| `type` | `type: "number"` |
| `contains` | `contains: "success"` |
| `startswith` | `startswith: "OK"` |
| `endswith` | `endswith: "_DONE"` |
| `matches` | `matches: "^[a-f0-9-]{36}$"` |
| `greater_than` | `greater_than: 0` |
| `less_than_or_equal` | `less_than_or_equal: 100` |
| `between` | `between: [18, 65]` |
| `length` | `length: 5` |
| `min_length` | `min_length: 1` |
| `empty` | `empty: true` |
| `exists` | `exists: false` |
| `contains_value` | `contains_value: "admin"` |
| `deep_equals` | `deep_equals: {key: "val"}` |

Only one matcher is active per assertion block (first one set wins).

## Examples

| File | What it demonstrates |
|------|---------------------|
| `examples/health_check.yaml` | Minimal GET + status check |
| `examples/pi.yaml` | Variables in URL, multi-step |
| `examples/assertions.yaml` | All 15 assertion matchers |
| `examples/conditional.yaml` | `run_if`, `skip_if_failed`, `skip` |
| `examples/retry.yaml` | Retry with backoff |
| `examples/variables.yaml` | Config and env variable substitution |

## Project structure

```
tango/
├── main.go                        # CLI entry point (cobra)
├── cmd/
│   └── validate.go                # Validate subcommand
├── internal/
│   ├── config/
│   │   ├── types.go               # All data structures
│   │   ├── parser.go              # YAML parsing + validation
│   │   └── parser_test.go
│   ├── executor/
│   │   ├── runner.go              # Step execution, retries, templates
│   │   ├── dependency.go          # Topological sort (Kahn's algorithm)
│   │   └── state.go               # Shared state between steps
│   ├── http/
│   │   ├── client.go              # HTTP client wrapper
│   │   └── client_test.go
│   ├── assertions/
│   │   ├── validator.go           # All 15 assertion matchers
│   │   └── validator_test.go
│   └── output/
│       └── text.go                # Terminal output formatting
├── examples/                      # Example workflow files
└── Makefile
```

## Makefile targets

```bash
make build    # Build the binary
make test     # Run all tests
make run      # Build + run health check example
make validate # Validate all examples
make lint     # Run go vet
make clean    # Remove binary
```

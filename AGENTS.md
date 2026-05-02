# AGENTS.md

## What is tango

Declarative API testing CLI. Define HTTP workflows in YAML, chain requests, capture response data, validate with JSONPath assertions, control flow with conditions, and retry with exponential backoff.

## Project layout

```
tango/
├── main.go                    # CLI entry point (cobra). var Version = "dev"
├── cmd/
│   └── validate.go            # `tango validate` subcommand
├── internal/
│   ├── config/
│   │   ├── types.go           # All data structs (Workflow, Step, AssertionSpec...)
│   │   ├── parser.go          # YAML parse + validate + applyDefaults
│   │   └── parser_test.go
│   ├── executor/
│   │   ├── runner.go          # Step execution, retry logic, template resolution
│   │   ├── dependency.go      # Topological sort (Kahn's algorithm)
│   │   └── state.go           # Shared state: captured vars, step statuses, ShouldSkip
│   ├── http/
│   │   ├── client.go          # HTTP client wrapper (Request/Response structs)
│   │   └── client_test.go
│   ├── assertions/
│   │   ├── validator.go       # 15 assertion matchers
│   │   └── validator_test.go
│   └── output/
│       └── text.go            # Terminal output formatting
├── examples/                  # Workflow YAML files
├── docs/                      # User documentation
├── scripts/
│   └── release.sh             # Bump version, commit, tag, push
├── nfpm.yaml                  # .deb package config
├── tango.spec                 # RPM spec (COPR)
├── .github/workflows/         # CI/CD
│   ├── release.yml            # Binary build + nfpm .deb + GitHub Release
│   └── copr-rpm.yml           # COPR RPM build + Homebrew formula update
├── Makefile
└── go.mod                     # Module: github.com/pc0stas/tango, Go 1.25
```

## Key dependencies

| Dependency | Usage |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/tidwall/gjson` | JSONPath querying for assertions and capture |
| `gopkg.in/yaml.v3` | YAML config parsing |

## How to build and test

```bash
make build         # Build local binary to ./tango
make build-all     # Cross-compile all platforms to ./build/
make test          # Run all tests (go test ./internal/...)
make run           # Build + run health_check example
make validate      # Build + validate all example YAMLs
make lint          # go vet ./...
make clean         # Remove binary and build/
```

## How it works (request flow)

1. `main.go` → cobra parses `tango test <file.yaml>`
2. `config.ParseWorkflow()` → YAML unmarshal → validate → applyDefaults (adds implicit depends_on from run_if/skip_if_failed/template refs)
3. `executor.NewExecutor(workflow)` → creates http client
4. `executor.Run()` → topological sort → execute steps in order
5. `executor.executeStep()`:
   - Resolve templates `{{ .var }}`, `{{ config.var }}`, `{{ steps.X.Y }}`, `{{ env.X }}`
   - Build HTTP request
   - Retry loop with exponential backoff (if retry_count > 0)
   - assertions.Validate() → check status, body assertions, response time
   - Capture json_path + response_body into ExecutionState
6. `output.FormatText()` → terminal output
7. Exit code 0 if all passed, 1 if any failed

## Commands

```bash
tango test <file.yaml>          # Execute workflow
tango test --dump <file.yaml>   # Execute with full request/response dump
tango validate <file.yaml>      # Validate YAML syntax only
tango completion <shell>        # Generate completions (bash/zsh/fish)
tango --version                 # Show version
```

## Code conventions

- **Imports**: stdlib first, blank line, third-party, blank line, internal packages
- **Internal http package**: imported as bare `http` (no alias needed, net/http not imported in those files)
- **Assertions package**: imported as `assertions`
- **Config package**: imported as `config`
- **Error wrapping**: use `fmt.Errorf("context: %w", err)` throughout
- **gjson paths**: no `$` prefix. Use `name`, `items.0.id`, `address.street`
- **YAML numbers**: parsed as `int` by yaml.v3. Assertion comparisons coerce to `float64` via `toFloat64()`
- **Booleans in assertions**: `Empty` and `Exists` are `*bool` to distinguish nil (not set) from false

### Template syntax

| Pattern | Source |
|---|---|
| `{{ .NAME }}` | Config variables (shortcut) |
| `{{ config.NAME }}` | Config variables (explicit) |
| `{{ steps.STEP.VAR }}` | Captured from previous step |
| `{{ env.NAME }}` | OS environment variable |

Templates are resolved in URL, headers, body, and assertion values (string fields only).

## Implicit depends_on

The parser automatically derives `depends_on` from:
- `run_if.previous_step` → adds dependency on that step
- `skip_if_failed: "X"` → adds dependency on step X
- Any `{{ steps.X.Y }}` ref in URL/body/headers → adds dependency on step X

No need to declare them manually in YAML.

## Release process

```bash
./scripts/release.sh vX.Y.Z
```

This bumps version in `main.go` + `tango.spec`, commits, pushes, and creates a Git tag. GitHub Actions then:
1. `release.yml` → cross-compile 5 platforms + nfpm .deb + GitHub Release
2. `copr-rpm.yml` → SRPM → COPR build (RFC RPM) → update homebrew-tango formula

## Distribution channels

| Channel | How |
|---|---|
| macOS | `brew tap pc0stas/tango && brew install tango` |
| Fedora | `dnf copr enable pablocostas/tango && dnf install tango` |
| Debian/Ubuntu | Download .deb from GitHub Release → `dpkg -i` |
| Binaries | Download from GitHub Release → `chmod +x` |

## Common issues

- **gjson paths**: Use `id` not `$.id`. gjson doesn't support JSONPath `$` prefix.
- **Number comparisons**: YAML `equals: 1` is `int(1)`, JSON response is `float64(1)`. The validator's `valuesEqual()` coerces both to float64.
- **COPR builds**: COPR has no network access. Source tarball must include `vendor/`. The `copr-rpm.yml` workflow runs `go mod vendor` before packaging.
- **Template resolution in assertions**: Only works for string fields (`equals`, `contains`, etc.). If `equals` value is a YAML string like `"{{ steps.X.Y }}"`, it gets resolved. Numeric template refs in assertions need to be strings (e.g. `equals: "{{ steps.X.id }}"`).

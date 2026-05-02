# Examples

Example workflows live in the `examples/` directory.

| File | What it demonstrates |
|---|---|
| `health_check.yaml` | Minimal GET request with status check |
| `pi.yaml` | Variables in URLs, multi-step checks against a homelab |
| `assertions.yaml` | All 15 assertion matchers in a single step |
| `conditional.yaml` | `run_if`, `skip_if_failed`, and `skip` flags |
| `retry.yaml` | Retry with exponential backoff on 5xx errors |
| `variables.yaml` | Config variable substitution (`.name` and `config.name` syntax) |
| `capture_demo.yaml` | Full capture flow: json_path, response_body, reuse in URL, headers, body, and assertions |
| `crud.yaml` | Capture `id` from list response and reuse in subsequent GETs |

## Run an example

```bash
tango test examples/capture_demo.yaml
tango test --dump examples/assertions.yaml   # Full request/response dump
```

## Validate an example

```bash
tango validate examples/conditional.yaml
```

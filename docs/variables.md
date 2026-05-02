# Variables

Tango supports template substitution in URLs, headers, request bodies, and even assertion values.

## Syntax

Templates use `{{ }}` delimiters and resolve just before each request.

| Syntax | Source | Example |
|---|---|---|
| `{{ .NAME }}` | Config variable | `{{ .host }}` |
| `{{ config.NAME }}` | Config variable (explicit) | `{{ config.host }}` |
| `{{ steps.STEP.VAR }}` | Captured from step | `{{ steps.login.token }}` |
| `{{ env.NAME }}` | Environment variable | `{{ env.API_KEY }}` |

Templates in assertion values (like `equals`) are resolved before validation.

## Config variables

Define in the workflow config:

```yaml
config:
  variables:
    host: "api.example.com"
    port: "8080"

steps:
  - request:
      url: "https://{{ .host }}:{{ .port }}/data"
```

## Capturing response data

Extract values from a response to use in later steps.

### JSONPath extraction

```yaml
- name: "login"
  request:
    url: "https://api.example.com/login"
  capture:
    json_path:
      token: "access_token"
      user_id: "user.id"
```

Values are extracted from the JSON response body using gjson path syntax. Then referenced as:

```yaml
- name: "get_profile"
  request:
    url: "https://api.example.com/users/{{ steps.login.user_id }}"
    headers:
      Authorization: "Bearer {{ steps.login.token }}"
```

### Full body capture

```yaml
- name: "save_data"
  request:
    url: "https://api.example.com/data/1"
  capture:
    response_body: "cached"
```

The entire response body string is stored and can be reused:

```yaml
- name: "replay"
  request:
    method: "POST"
    url: "https://api.example.com/data"
    body: "{{ steps.save_data.cached }}"
```

## Implicit dependencies

Steps that reference captured variables (`{{ steps.STEP.VAR }}`) automatically gain an implicit `depends_on` on the source step. You don't need to declare it manually.

## Environment variables

```yaml
- request:
    url: "https://{{ env.API_HOST }}/data"
    headers:
      X-API-Key: "{{ env.SECRET_KEY }}"
```

# Conditional Execution

Control which steps run based on the outcome of previous steps.

## skip

Always skip a step:

```yaml
- name: "debug_step"
  skip: true
  request:
    url: "https://api.example.com/debug"
```

## skip_if_failed

Skip this step if the referenced step failed:

```yaml
- name: "rollback"
  skip_if_failed: "deploy"
  request:
    url: "https://api.example.com/rollback"
```

Only triggers when the referenced step status is `"failed"`. Does nothing if the step passed or was skipped.

## run_if

Only execute this step if the referenced step finished with a specific status:

```yaml
- name: "verify_deploy"
  run_if:
    previous_step: "deploy"
    status: "passed"
  request:
    url: "https://api.example.com/health"
```

```yaml
- name: "cleanup"
  run_if:
    previous_step: "deploy"
    status: "failed"
  request:
    url: "https://api.example.com/cleanup"
```

Valid statuses: `"passed"`, `"failed"`, `"skipped"`.

## Implicit dependencies

All conditional fields (`run_if.previous_step`, `skip_if_failed`) automatically add a `depends_on` to ensure the referenced step runs first. No need to duplicate.

## Combined example

```yaml
steps:
  - name: "create"
    request:
      method: "POST"
      url: "https://api.example.com/users"
    capture:
      json_path:
        user_id: "id"

  - name: "verify"
    run_if:
      previous_step: "create"
      status: "passed"
    request:
      url: "https://api.example.com/users/{{ steps.create.user_id }}"
    expect:
      status: 200

  - name: "cleanup"
    skip_if_failed: "verify"
    request:
      method: "DELETE"
      url: "https://api.example.com/users/{{ steps.create.user_id }}"
```

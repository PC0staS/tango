# Quick Start

## Install

```bash
# macOS
brew tap pc0stas/tango && brew install tango

# Fedora / RHEL
sudo dnf copr enable pc0stas/tango && sudo dnf install tango

# Debian / Ubuntu
wget https://github.com/pc0stas/tango/releases/latest/download/tango_amd64.deb
sudo dpkg -i tango_amd64.deb

# From source
git clone https://github.com/pc0stas/tango && cd tango && make build
```

## Your first workflow

Create `health.yaml`:

```yaml
name: "my-first-test"
config:
  timeout_default: 5s
  stop_on_error: false

steps:
  - name: "ping"
    request:
      method: "GET"
      url: "https://httpbin.org/status/200"
    expect:
      status: 200
```

Run it:

```bash
tango test health.yaml
```

## Commands

| Command | Description |
|---|---|
| `tango test <file.yaml>` | Execute a workflow |
| `tango test --dump <file.yaml>` | Execute with full request/response dump |
| `tango validate <file.yaml>` | Validate YAML syntax and structure |
| `tango --version` | Show version |
| `tango completion <shell>` | Generate shell completions |

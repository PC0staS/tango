# Tango

Declarative API testing CLI. Define HTTP workflows in YAML, chain requests, capture data, and validate responses.

## Install

```bash
# macOS
brew tap pc0stas/tango && brew install tango

# Fedora / RHEL
sudo dnf copr enable pablocostas/tango && sudo dnf install tango

# Debian / Ubuntu
wget https://github.com/pc0stas/tango/releases/latest/download/tango_amd64.deb
sudo dpkg -i tango_amd64.deb

# Arch Linux (AUR)
yay -S tango-cli

# From source
git clone https://github.com/pc0stas/tango && cd tango && make build
```

## Quick start

```bash
tango test examples/health_check.yaml
```

## Shell completions

Package managers (brew, dnf, dpkg) install completions automatically. If you installed from source or a raw binary, enable them manually:

```bash
# bash — add to ~/.bashrc
source <(tango completion bash)

# zsh — add to ~/.zshrc
source <(tango completion zsh)

# fish
tango completion fish > ~/.config/fish/completions/tango.fish
```

## Documentation

See [docs/](docs/) for full guides on workflows, assertions, variables, conditional execution, retries, and more.

## Contributing

```bash
go test ./...        # Run tests
make validate        # Validate all examples
make build-all       # Cross-compile
```

PRs welcome. Open an issue before starting large changes.

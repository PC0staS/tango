Name:           tango
Version:        1.0.6
Release:        1%{?dist}
Summary:        Declarative API testing CLI
License:        MIT
URL:            https://github.com/pc0stas/tango
Source0:        tango-%{version}.tar.gz

BuildRequires:  golang >= 1.25
BuildRequires:  git

%description
Declarative API testing CLI. Define HTTP workflows in YAML,
chain requests, capture data, validate responses with JSONPath
assertions, and run conditional steps with retry logic.

%prep
%setup -q -n tango-%{version}

%build
go build -ldflags "-X main.Version=%{version}" -o tango .

# Generate shell completions
mkdir -p completions
./tango completion bash > completions/tango.bash
./tango completion zsh  > completions/_tango
./tango completion fish > completions/tango.fish

%install
install -Dm755 tango %{buildroot}%{_bindir}/tango
install -Dm644 completions/tango.bash %{buildroot}%{_datadir}/bash-completion/completions/tango
install -Dm644 completions/_tango      %{buildroot}%{_datadir}/zsh/site-functions/_tango
install -Dm644 completions/tango.fish  %{buildroot}%{_datadir}/fish/vendor_completions.d/tango.fish

%files
%{_bindir}/tango
%{_datadir}/bash-completion/completions/tango
%{_datadir}/zsh/site-functions/_tango
%{_datadir}/fish/vendor_completions.d/tango.fish
%license LICENSE
%doc README.md

%changelog
* Sat May 02 2026 Pablo <pablo@example.com> - 0.1.0-1
- Initial release

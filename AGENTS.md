# opentofu-kms-stackit

An OpenTofu external key provider: OpenTofu launches this binary, sends stored
metadata on stdin, and reads a data key wrapped by STACKIT KMS from stdout.

## Invariants

- **stdout carries the protocol only**: the header line, then the JSON response.
  Send every diagnostic to stderr through `log`.
- **Secrets enter through environment variables only.** Paths and IDs may be
  flags; key material, tokens and private keys stay out of the flag set, because
  flags land in committed OpenTofu configuration, process listings and usage
  errors.
- **The compatibility surface is the command line and the stored metadata**:
  flag names, environment variable names, the protocol, and the `ciphertext` and
  `key_version` metadata fields. Those two fields are persisted inside users'
  state files, so renaming or retyping them makes existing state undecryptable.
  Go symbols under `internal/` are free to change.
- **Encrypt with the configured version, decrypt with the stored one.** The
  version in the input metadata decides which key version unwraps; the
  configured or latest-active version only wraps new data keys.
- **Input metadata is untrusted.** It comes from the state file; validate it
  before it reaches a KMS call.

## Making changes

- A new setting goes through `envOrDefault` in `internal/config` (flag, then
  environment variable, then default). Required settings also go into
  `validate`, so a missing value fails before any API call.
- KMS access goes through the `keyManagementClient` interface in
  `internal/keyprovider/provider.go`; tests use the mock in `mock_test.go`
  against that interface, since the SDK's request types cannot be mocked.
- `install.sh` is POSIX `sh` and is linted with ShellCheck in CI. It rebuilds
  asset names from `name_template` in `.goreleaser.yaml`; change both in the
  same commit.
- The golangci-lint version is pinned in both `.mise.toml` and
  `.github/workflows/ci.yaml`; bump them together.
- GitHub Actions are pinned to a full commit SHA with the tag in a trailing
  comment; keep that form when adding or updating one.
- Documentation describes current behaviour. History belongs in git.
- Changed behaviour comes with unit tests.

## Done

A change is done when `task lint` and `task test` pass and `go mod tidy` leaves
`go.mod` and `go.sum` untouched. Tool versions come from mise (`mise install`);
`task -l` lists every task.

## Further reading

- [CONTRIBUTING.md](CONTRIBUTING.md): local setup, the git hook, pull request
  expectations. Read before opening a pull request.
- [RELEASE.md](RELEASE.md): tagging, the release pipeline, macOS signing,
  versioning rules. Read before touching `.goreleaser.yaml`, the release
  workflow or a tag.
- [README.md](README.md): user-facing flags, environment variables and usage.
  Update it when the command line changes.

# Contributing

Thanks for taking the time to contribute. This document covers the local setup,
the checks your change has to pass, and what a pull request should contain.

For publishing a release, see [RELEASE.md](RELEASE.md).

## Table of Contents

- [Getting started](#getting-started)
- [The git hook](#the-git-hook)
- [Running the checks](#running-the-checks)
- [Project layout](#project-layout)
- [Conventions](#conventions)
- [Pull requests](#pull-requests)

## Getting started

Tool versions are pinned in `.mise.toml` ([mise](https://mise.jdx.dev)) and tasks
live in `taskfile.yaml` ([Task](https://taskfile.dev)).

mise provides the toolchain, including `task` itself, so install it first:

```shell
brew install mise                # or see https://mise.jdx.dev/installing-mise.html
eval "$(mise activate zsh)"      # add this line to ~/.zshrc as well
```

Then, once per clone, trust the config and install everything:

```shell
mise trust
mise install
task setup
```

`mise install` gives you go, task, golangci-lint, goreleaser and pre-commit at the
pinned versions. `task setup` re-runs `mise install` and installs the git hook via
`pre-commit install`.

Run `task -l` to see every available command.

## The git hook

The hook defined in `.pre-commit-config.yaml` runs on every `git commit` that
touches Go files:

| Hook               | Command                   | Effect                              |
| ------------------ | ------------------------- | ----------------------------------- |
| golangci-lint fmt  | `golangci-lint fmt`       | Applies gofmt and goimports         |
| golangci-lint      | `golangci-lint run --fix` | Lints and type-checks, fixes what it can |
| go test            | `go test ./...`           | Runs the test suite                 |
| go mod tidy        | `go mod tidy`             | Keeps `go.mod` and `go.sum` in sync |

A hook that rewrites a file fails that commit on purpose. Review the changes,
`git add` them, and commit again.

To bypass the hook for a single commit, use `git commit --no-verify`.

## Running the checks

Linter selection and exclusions live in `.golangci.yml`.

```shell
task lint          # golangci-lint run
task fmt           # golangci-lint fmt
task test          # go test ./...
task hooks:run     # all hooks, staged files only
task hooks:run:all # all hooks, whole repo
```

CI runs the same checks on every pull request, plus a snapshot build of all
release targets. See `.github/workflows/ci.yaml`.

## Project layout

```
cmd/opentofu-kms-stackit/   the binary, protocol wiring and flag handling
internal/config/            flag and env var parsing
internal/keyprovider/       the key provider protocol and KMS calls
install.sh                  installer for the published release archives
```

Everything outside `cmd/` lives in `internal/`, so no external module can import
it. That keeps compatibility promises about the command line surface, meaning
flags, environment variables and the key provider protocol, rather than about Go
symbols.

## Conventions

- **stdout belongs to the protocol.** The binary writes the protocol header and
  the JSON response to stdout, and every diagnostic goes to stderr through `log`.
  Any other write to stdout corrupts what OpenTofu parses.
- **Secrets come from the environment only.** Service account keys, tokens and
  private keys are read from environment variables and are deliberately absent
  from the flag set, because flags end up in committed OpenTofu configuration and
  in process listings. Paths and IDs are fine as flags.
- **New settings follow `envOrDefault`**: the flag wins, then the environment
  variable, then the default. Add required ones to `validate` so a missing value
  fails before any API call.
- Documentation describes current behaviour. History belongs in git.

## Pull requests

- Link the issue your change relates to, or describe the problem it solves.
- Add or update unit tests for behaviour you change.
- Make sure `task lint` and `task test` pass, since CI checks both.
- Keep the change focused. Unrelated cleanups are easier to review separately.

The pull request template lists the same points as a checklist.

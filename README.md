# opentofu-kms-stackit

[![OpenTofu](https://img.shields.io/badge/OpenTofu-1.12+-FFDA18?logo=opentofu&logoColor=black)](https://opentofu.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

An [OpenTofu](https://opentofu.org) [external key provider](https://opentofu.org/docs/language/state/encryption/#external-experimental) backed by [STACKIT Key Management Service](https://docs.stackit.cloud/products/security/kms/), for encrypting OpenTofu state and plan files.

Each time OpenTofu writes state, this binary asks STACKIT KMS to wrap a freshly generated data key and returns only that wrapped ciphertext for OpenTofu to store; the plaintext key itself never touches disk. On the next read, OpenTofu hands the stored ciphertext back to this binary, which asks STACKIT KMS to unwrap it.

You will not invoke this program directly: OpenTofu launches it as a subprocess and talks to it over the [external key provider protocol](https://opentofu.org/docs/language/state/encryption/#writing-an-external-key-provider) (JSON over stdin/stdout).

## Table of Contents

- [Installation](#installation)
- [Configuration](#configuration)
  - [Authentication](#authentication)
  - [Key selection](#key-selection)
- [Usage](#usage)
- [Development](#development)
- [Related links](#related-links)

## Installation

OpenTofu launches this binary via the `command` array in your `key_provider "external"` block (see [Usage](#usage)).
Either put it on your `PATH` and reference it by name, or point `command` at its full path directly.

Install with Go:

```sh
go install github.com/stackitcloud/opentofu-kms-stackit/cmd/opentofu-kms-stackit@latest
```

Or clone and build it yourself. This requires [Task](https://taskfile.dev/installation/) to be installed first:

```sh
git clone https://github.com/stackitcloud/opentofu-kms-stackit.git
cd opentofu-kms-stackit
task install # installs to /usr/local/bin (override with PREFIX=...)
```

## Configuration

Configuration is split into two groups:

1. **Authentication** - resolved automatically by `stackit-sdk-go`'s own credential chain, the same one the [STACKIT Terraform provider](https://registry.terraform.io/providers/stackitcloud/stackit/latest/docs) uses. No configuration needed for the common case.
2. **Key selection** - which key to use, passed as command-line flags in the OpenTofu `command` block (see [Usage](#usage)).

### Authentication

Credentials are resolved from `STACKIT_*` environment variables or `~/.stackit/credentials.json`, in the order documented for [`stackit-sdk-go`](https://github.com/stackitcloud/stackit-sdk-go). Common variables:

| Environment variable               | Description                         |
| ---------------------------------- | ----------------------------------- |
| `STACKIT_SERVICE_ACCOUNT_KEY_PATH` | Path to a service account key file  |
| `STACKIT_SERVICE_ACCOUNT_KEY`      | Service account key JSON            |
| `STACKIT_PRIVATE_KEY_PATH`         | Path to the matching private key    |
| `STACKIT_SERVICE_ACCOUNT_TOKEN`    | Static service account access token |

`STACKIT_SERVICE_ACCOUNT_KEY_PATH` and `STACKIT_PRIVATE_KEY_PATH` also have matching `--service-account-key-path`/`--private-key-path` flags, and `--endpoint` overrides the KMS API endpoint. The raw secret values (`STACKIT_SERVICE_ACCOUNT_KEY`, `STACKIT_SERVICE_ACCOUNT_TOKEN`, `STACKIT_PRIVATE_KEY`) are environment-variable-only, intentionally not exposed as flags: OpenTofu captures this program's stderr as diagnostics, and a flag's default value gets echoed back on a usage error, so a secret passed as a flag default risks ending up in your logs. Command-line arguments in general are also visible to other processes on the same machine.

### Key selection

| Flag            | Required | Default               | Environment variable      | Description                                                             |
| --------------- | -------- | --------------------- | ------------------------- | ----------------------------------------------------------------------- |
| `--project-id`  | yes      | -                     | `STACKIT_KMS_PROJECT_ID`  | STACKIT project UUID                                                    |
| `--region`      | no       | `eu01`                | `STACKIT_KMS_REGION`      | STACKIT region the key ring is located in                               |
| `--key-ring-id` | yes      | -                     | `STACKIT_KMS_KEY_RING_ID` | Key ring UUID                                                           |
| `--key-id`      | yes      | -                     | `STACKIT_KMS_KEY_ID`      | Key UUID, purpose `symmetric_encrypt_decrypt`, algorithm `aes_256_gcm`  |
| `--key-version` | no       | latest active version | `STACKIT_KMS_KEY_VERSION` | Pins the key version; auto-resolved and stored per state file otherwise |

Each flag falls back to its environment variable when not given; the flag wins when both are set. The generated data key is always 32 bytes (AES-256), matching one of `aes_gcm`'s three accepted key sizes; there is no option to change it.

## Usage

Add an `encryption` block to your OpenTofu configuration, pointing the external key provider at this binary:

```hcl
terraform {
  encryption {
    key_provider "external" "stackit" {
      command = [
        "opentofu-kms-stackit",
        "--project-id", "00000000-0000-0000-0000-000000000000",
        "--key-ring-id", "11111111-1111-1111-1111-111111111111",
        "--key-id", "22222222-2222-2222-2222-222222222222",
      ]
    }

    method "aes_gcm" "stackit" {
      keys = key_provider.external.stackit
    }

    state {
      method = method.aes_gcm.stackit
    }
    plan {
      method = method.aes_gcm.stackit
    }
  }
}
```

## Development

Tool versions are pinned in `.mise.toml` ([mise](https://mise.jdx.dev)) and tasks live in `taskfile.yaml` ([Task](https://taskfile.dev)).

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

`mise install` gives you go, task, golangci-lint and pre-commit at the pinned versions. `task setup` re-runs `mise install` and installs the git hook via `pre-commit install`. The hook defined in `.pre-commit-config.yaml` runs on every `git commit` when Go files change:

| Hook               | Command                  | Effect                                          |
| ------------------ | ------------------------ | ----------------------------------------------- |
| golangci-lint fmt  | `golangci-lint fmt`      | Applies gofmt and goimports                     |
| golangci-lint      | `golangci-lint run --fix` | Lints and type-checks, fixes what it can       |
| go test            | `go test ./...`          | Runs the test suite                             |
| go mod tidy        | `go mod tidy`            | Keeps `go.mod` and `go.sum` in sync             |

A hook that rewrites a file fails the commit, so review the changes and `git add` them before committing again.

Linter selection and exclusions live in `.golangci.yml`. Run the checks without committing:

```shell
task lint          # golangci-lint run
task fmt           # golangci-lint fmt
task test          # go test ./...
task hooks:run     # all hooks, staged files only
task hooks:run:all # all hooks, whole repo
```

To bypass the hook for a single commit, use `git commit --no-verify`.

## Related links

- STACKIT Key Management Service docs: https://docs.stackit.cloud/products/security/kms/
- OpenTofu state/plan encryption docs: https://opentofu.org/docs/language/state/encryption/

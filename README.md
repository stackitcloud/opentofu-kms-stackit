# opentofu-kms-stackit

[![CI](https://github.com/stackitcloud/opentofu-kms-stackit/actions/workflows/ci.yaml/badge.svg)](https://github.com/stackitcloud/opentofu-kms-stackit/actions/workflows/ci.yaml)
[![OpenTofu](https://img.shields.io/badge/OpenTofu-1.12+-FFDA18?logo=opentofu&logoColor=black)](https://opentofu.org/)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

An [OpenTofu](https://opentofu.org) [external key provider](https://opentofu.org/docs/language/state/encryption/#external-experimental) backed by [STACKIT Key Management Service](https://docs.stackit.cloud/products/security/kms/), for encrypting OpenTofu state and plan files.

Each time OpenTofu writes state, this binary asks STACKIT KMS to wrap a freshly generated data key and returns only that wrapped ciphertext for OpenTofu to store; the plaintext key itself never touches disk. On the next read, OpenTofu hands the stored ciphertext back to this binary, which asks STACKIT KMS to unwrap it.

You will not invoke this program directly: OpenTofu launches it as a subprocess and talks to it over the [external key provider protocol](https://opentofu.org/docs/language/state/encryption/#writing-an-external-key-provider) (JSON over stdin/stdout).

## Table of Contents

- [Installation](#installation)
  - [Install script](#install-script)
  - [Pre-compiled binary](#pre-compiled-binary)
  - [From source](#from-source)
- [Configuration](#configuration)
  - [Authentication](#authentication)
  - [Key selection](#key-selection)
- [Usage](#usage)
- [Development](#development)
- [Related links](#related-links)

## Installation

OpenTofu launches this binary via the `command` array in your `key_provider "external"` block (see [Usage](#usage)).
Either put it on your `PATH` and reference it by name, or point `command` at its full path directly.

Prebuilt binaries are published on the [Releases](https://github.com/stackitcloud/opentofu-kms-stackit/releases) page for Linux and macOS, on both amd64 and arm64.

### Install script

```sh
curl -fsSL https://raw.githubusercontent.com/stackitcloud/opentofu-kms-stackit/main/install.sh | sh
```

The script downloads the release archive for your platform, verifies it against `checksums.txt` and installs the binary to `/usr/local/bin`.
It asks for `sudo` only when that directory is not writable.

Two environment variables change what it does:

| Variable      | Default        | Effect                                    |
| ------------- | -------------- | ----------------------------------------- |
| `VERSION`     | latest release | Release tag to install, e.g. `v0.1.0`     |
| `INSTALL_DIR` | `/usr/local/bin` | Directory to install the binary into    |

To pin a version or install without root:

```sh
curl -fsSL https://raw.githubusercontent.com/stackitcloud/opentofu-kms-stackit/main/install.sh -o install.sh
VERSION=v0.1.0 INSTALL_DIR="$HOME/.local/bin" sh install.sh
```

Check what you installed:

```sh
opentofu-kms-stackit -version
# opentofu-kms-stackit 0.1.0 (commit 9fceb02..., built 2026-08-28T10:22:56Z)
```

### Pre-compiled binary

1. Download the archive for your operating system and CPU architecture from the [Releases](https://github.com/stackitcloud/opentofu-kms-stackit/releases) page
2. Verify it against `checksums.txt`:

   ```sh
   sha256sum --check --ignore-missing checksums.txt
   ```

3. Extract it and move `opentofu-kms-stackit` to a directory on your `PATH`

### From source

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

Setup, the git hook and the checks a change has to pass are documented in
[CONTRIBUTING.md](CONTRIBUTING.md). Release process is documented in
[RELEASE.md](RELEASE.md). Participation is covered by the
[Code of Conduct](CODE_OF_CONDUCT.md).

## Related links

- STACKIT Key Management Service docs: https://docs.stackit.cloud/products/security/kms/
- OpenTofu state/plan encryption docs: https://opentofu.org/docs/language/state/encryption/

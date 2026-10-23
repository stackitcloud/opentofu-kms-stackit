# Release

A release is published by pushing a semver tag. The git tag is the only source
of the version, there is no version constant to bump.

## Creating a release

1. Check out the latest `main` and make sure CI is green.
2. Verify the release config and build the archives locally:

   ```shell
   task release:check    # validates .goreleaser.yaml
   task release:snapshot # writes ./dist, publishes nothing
   ```

3. Create and push the tag:

   ```shell
   git tag v0.1.0
   git push origin v0.1.0
   ```

4. Watch [the release workflow](../../actions/workflows/release.yaml) and check
   the result on the [releases page](../../releases).

## What the pipeline does

`.github/workflows/release.yaml` runs [GoReleaser](https://goreleaser.com) on
tags matching `v[0-9]+.[0-9]+.[0-9]+` and `v[0-9]+.[0-9]+.[0-9]+-*`, which:

- builds binaries with CGO disabled for linux and darwin on amd64 and arm64
- signs the darwin binaries with the Developer ID certificate and notarizes them
  with Apple, see [macOS signing](#macos-signing)
- packs each one with `LICENSE` and `README.md` into a `tar.gz`
- writes `checksums.txt` over all archives
- creates the GitHub release with notes generated from the commits

The version, commit and build date are stamped into the binary through ldflags,
so `opentofu-kms-stackit -version` reports the tag it was built from. Builds from
source report `dev`.

## macOS signing

GoReleaser signs and notarizes the darwin binaries through
[quill](https://github.com/anchore/quill), so the job runs on Linux and needs no
keychain. macOS Gatekeeper blocks binaries that are downloaded through a browser
and not notarized. Snapshot builds skip signing. A tagged release fails if the
certificate is missing.

The release job reads these secrets from the `signing` GitHub environment:

| Secret                   | Value                                                   |
| ------------------------ | ------------------------------------------------------- |
| `MACOS_SIGN_P12`         | base64 of the "Developer ID Application" `.p12`         |
| `MACOS_SIGN_PASSWORD`    | password of the `.p12`                                  |
| `MACOS_NOTARY_KEY`       | base64 of the App Store Connect API key `.p8`           |
| `MACOS_NOTARY_KEY_ID`    | ID of the API key, also part of the `.p8` file name     |
| `MACOS_NOTARY_ISSUER_ID` | issuer UUID shown above the App Store Connect key list  |

Without the three notary secrets, GoReleaser signs the binaries but skips
notarization.

To set or rotate the secrets:

```shell
base64 -i developer_id_application.p12 | gh secret set MACOS_SIGN_P12 --env signing
gh secret set MACOS_SIGN_PASSWORD --env signing
base64 -i AuthKey_XXXXXXXXXX.p8 | gh secret set MACOS_NOTARY_KEY --env signing
gh secret set MACOS_NOTARY_KEY_ID --env signing
gh secret set MACOS_NOTARY_ISSUER_ID --env signing
```

To check a released binary on a Mac:

```shell
codesign --verify --strict --verbose=2 opentofu-kms-stackit
spctl --assess --type install --verbose opentofu-kms-stackit
```

## Pre-releases

A tag with a prerelease suffix such as `v0.1.0-rc1` is marked as a pre-release on
GitHub. `install.sh` without a `VERSION` resolves the latest stable release, so
pre-releases reach only the people who ask for them by name:

```shell
VERSION=v0.1.0-rc1 sh install.sh
```

## Versioning

While the project is pre-1.0, breaking changes bump the minor version and fixes
bump the patch. The compatibility surface is the command line: flag names,
environment variables and the key provider protocol. Internal Go packages are not
part of it.

Two constraints worth knowing:

- `install.sh` reconstructs asset names from `name_template` in
  `.goreleaser.yaml`. Changing the template means changing the script in the same
  commit, or downloads break.
- At `v2.0.0` the module path has to gain a `/v2` suffix
  (`module github.com/stackitcloud/opentofu-kms-stackit/v2`), with the internal
  imports updated in the same commit. Otherwise `go install ...@v2` fails.
  Downloads through `install.sh` and the releases page are unaffected.

## Fixing a bad release

Delete the tag and the release, then tag again:

```shell
gh release delete v0.1.0 --yes
git push --delete origin v0.1.0
git tag -d v0.1.0
```

Do this only for a release nobody consumed yet. Once people have downloaded it,
publish a new patch version instead.

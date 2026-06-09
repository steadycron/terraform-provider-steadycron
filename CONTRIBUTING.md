# Contributing to terraform-provider-steadycron

## Development setup

```bash
git clone https://github.com/steadycron/terraform-provider-steadycron.git
cd terraform-provider-steadycron
go mod tidy
make build
```

Requires Go 1.22+.

## Running tests

### Unit tests

```bash
make test
```

### Acceptance tests

Acceptance tests provision and destroy real SteadyCron resources against the production API.

```bash
export STEADYCRON_API_KEY=sc_your_full_scope_key
make testacc
```

Tests auto-skip when `STEADYCRON_API_KEY` is absent. Each test creates and tears down its own
resources. Use a disposable account (not production) where possible.

## Local provider override

To test the provider locally with a real Terraform config, create a `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "steadycron/steadycron" = "/path/to/terraform-provider-steadycron"
  }
  direct {}
}
```

Build and install:

```bash
make install   # installs to GOPATH/bin
```

Then `terraform init` is not needed — Terraform picks up the dev override directly.

## Generating docs

Provider documentation is generated from the schema `MarkdownDescription` fields by
[`tfplugindocs`](https://github.com/hashicorp/terraform-plugin-docs):

```bash
make docs
```

Commit the updated `docs/` directory. CI enforces that committed docs match generated output.

## Submitting changes

1. Fork the repository and create a feature branch.
2. Make changes; add or update tests.
3. Run `make vet test lint docs` and ensure all pass.
4. Open a pull request against `main`.
5. All acceptance tests must pass for merges to `main`.

## Release process

Releases are automated via [GoReleaser](https://goreleaser.com/) and triggered by pushing a Git tag.

### One-time setup (repo admin only)

1. **Generate a GPG signing key** for the Terraform Registry:
   ```bash
   gpg --batch --gen-key <<EOF
   %no-protection
   Key-Type: RSA
   Key-Length: 4096
   Subkey-Type: RSA
   Subkey-Length: 4096
   Name-Real: SteadyCron
   Name-Email: releases@steadycron.com
   Expire-Date: 0
   EOF
   ```
2. Export the public key and [register it in the Terraform Registry](https://registry.terraform.io/sign-in) under the `steadycron` namespace.
3. Export the private key and store it as the `GPG_PRIVATE_KEY` GitHub Actions secret; store the passphrase as `PASSPHRASE`.
4. Connect this repository in the Terraform Registry at **registry.terraform.io → Publish → Provider → steadycron/terraform-provider-steadycron**.

### Cutting a release

```bash
git tag v0.2.0
git push origin v0.2.0
```

The `release.yml` workflow:
1. Imports the GPG key.
2. Runs GoReleaser: builds cross-platform binaries, signs `SHA256SUMS`, creates a GitHub Release.
3. The Terraform Registry polls the GitHub Release and publishes the new version automatically.

### Post-release checklist

- [ ] Verify the release appears at `registry.terraform.io/providers/steadycron/steadycron`.
- [ ] Install in a clean project: `terraform init` with `source = "steadycron/steadycron"`.
- [ ] Update `CHANGELOG.md` with the release date.

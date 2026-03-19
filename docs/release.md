# Release Guide

## Versioning

- Follow semantic version tags: `vMAJOR.MINOR.PATCH`
- Use Git tags as the release source of truth
- GoReleaser injects `version`, `commit`, and `date` into the binary at build time

## Local verification

```bash
go test ./...
go build ./cmd/mynamr
goreleaser release --snapshot --clean
```

## Create a release tag

### Bash

```bash
./scripts/release-tag.sh v0.1.0
```

### PowerShell

```powershell
./scripts/release-tag.ps1 -Version v0.1.0
```

Both scripts validate the version format, require a clean working tree, run tests, and create an annotated tag only on success.

## Publish flow

1. Commit your release-ready changes
2. Push the release commit if it is not on GitHub yet
3. Push the created tag
4. GitHub Actions runs `.github/workflows/release.yml`
5. GoReleaser publishes archives and `checksums.txt`

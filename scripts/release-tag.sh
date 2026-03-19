#!/usr/bin/env bash

set -euo pipefail

VERSION="${1:-}"

if [[ -z "${VERSION}" ]]; then
  echo "usage: ./scripts/release-tag.sh v0.1.0"
  exit 1
fi

if [[ ! "${VERSION}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "error: version must match vMAJOR.MINOR.PATCH"
  exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
  echo "error: working tree is dirty; commit or stash changes first"
  exit 1
fi

if git rev-parse "${VERSION}" >/dev/null 2>&1; then
  echo "error: tag ${VERSION} already exists"
  exit 1
fi

echo "running tests before tagging ${VERSION}..."
go test ./...

echo "creating annotated tag ${VERSION}..."
git tag -a "${VERSION}" -m "release: ${VERSION}"

cat <<EOF
tag created locally: ${VERSION}

next steps:
  git push origin ${VERSION}

if the release commit is not on GitHub yet, push your branch first.
when the tag reaches GitHub, .github/workflows/release.yml will run GoReleaser.
EOF

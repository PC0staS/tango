#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 vX.Y.Z"
  exit 1
fi

RAW_VER="$1"
TAG_VER="$RAW_VER"
NO_V_VER="${RAW_VER#v}"

if [[ ! "$RAW_VER" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Invalid version format: $RAW_VER"
  echo "Expected vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH"
  exit 2
fi

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

# Clean up any stale backups from previous aborted runs
rm -f main.go.bak tango.spec.bak PKGBUILD.bak

if [ -n "$(git status --porcelain)" ]; then
  echo "Working tree not clean. Commit or stash changes first."
  git status --porcelain
  exit 3
fi

CURRENT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
echo "Current branch: $CURRENT_BRANCH"

if git rev-parse "refs/tags/$TAG_VER" >/dev/null 2>&1; then
  echo "Tag $TAG_VER already exists. Aborting."
  exit 4
fi

echo "Updating version to $NO_V_VER (tag: $TAG_VER)"

sed -E -i.bak "s/^(var Version = \").*(\")/\1${NO_V_VER}\2/" main.go
sed -E -i.bak "s/^(Version:)[[:space:]]+.*/Version:        ${NO_V_VER}/" tango.spec
sed -E -i.bak "s/^pkgver=.*/pkgver=${NO_V_VER}/" PKGBUILD

echo "Files updated. Showing git diff for review:"
git --no-pager diff -- main.go tango.spec PKGBUILD || true

read -p "Continue and commit changes? [y/N] " CONFIRM
if [[ "$CONFIRM" != "y" && "$CONFIRM" != "Y" ]]; then
  echo "Aborted by user. Restoring backups."
  mv -f main.go.bak main.go || true
  mv -f tango.spec.bak tango.spec || true
  mv -f PKGBUILD.bak PKGBUILD || true
  exit 5
fi

git add main.go tango.spec PKGBUILD
git commit -m "Release ${TAG_VER}"
git push origin "$CURRENT_BRANCH"

git tag -a "${TAG_VER}" -m "Release ${TAG_VER}"
git push origin "${TAG_VER}"

rm -f main.go.bak tango.spec.bak PKGBUILD.bak

echo "Done. Release ${TAG_VER} created and pushed."
echo "GitHub Actions will now build and publish binaries, .rpm, .deb, and AUR."

#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
VERSION_FILE="$REPO_ROOT/VERSION"

EXTENSIONS=(
  extensions/xconfig
  extensions/xgrpc
  extensions/xlog
  extensions/xopenapi
  extensions/xotel
  extensions/xvalidator
)

usage() {
  echo "Usage: $0 {patch|minor|major}"
  exit 1
}

if [ $# -ne 1 ]; then
  usage
fi

BUMP="$1"
case "$BUMP" in
  patch|minor|major) ;;
  *) usage ;;
esac

if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is dirty. Commit or stash changes first."
  exit 1
fi

CURRENT="$(cat "$VERSION_FILE" | tr -d '[:space:]')"
if ! echo "$CURRENT" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
  echo "Error: VERSION file does not contain a valid semver (found: '$CURRENT')"
  exit 1
fi

IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT"
case "$BUMP" in
  patch) PATCH=$((PATCH + 1)) ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
esac

NEW_VERSION="$MAJOR.$MINOR.$PATCH"

echo "Current version: $CURRENT"
echo "New version:     $NEW_VERSION"
echo ""
read -rp "Tag and push $NEW_VERSION? [y/N] " CONFIRM
if [[ ! "$CONFIRM" =~ ^[yY]$ ]]; then
  echo "Aborted."
  exit 0
fi

echo "$NEW_VERSION" > "$VERSION_FILE"

git add "$VERSION_FILE"
git commit -m "chore: bump version to $NEW_VERSION"

git tag "v$NEW_VERSION"

for EXT in "${EXTENSIONS[@]}"; do
  if [ -f "$REPO_ROOT/$EXT/go.mod" ]; then
    git tag "$EXT/v$NEW_VERSION"
  fi
done

echo ""
echo "Tags created locally. Run the following to push:"
echo ""
echo "  git push origin main"
echo "  git push origin --tags"
echo ""
echo "To undo (before push):"
echo "  git tag -d v$NEW_VERSION"
for EXT in "${EXTENSIONS[@]}"; do
  if [ -f "$REPO_ROOT/$EXT/go.mod" ]; then
    echo "  git tag -d $EXT/v$NEW_VERSION"
  fi
done
echo "  git reset --soft HEAD~1"

#!/usr/bin/env bash
set -euo pipefail

# Usage: ./scripts/release.sh [patch|minor|major]
# Bumps version, updates CHANGELOG, commits, tags, and pushes.

BUMP="${1:-patch}"
PLUGIN_JSON=".claude-plugin/plugin.json"
CHANGELOG="CHANGELOG.md"

# --- Read current version ---
CURRENT=$(grep '"version"' "$PLUGIN_JSON" | sed 's/.*: *"\(.*\)".*/\1/')
IFS='.' read -r MAJOR MINOR PATCH <<< "$CURRENT"

case "$BUMP" in
  patch) PATCH=$((PATCH + 1)) ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  *) echo "Usage: $0 [patch|minor|major]"; exit 1 ;;
esac

NEW="${MAJOR}.${MINOR}.${PATCH}"
TAG="v${NEW}"

echo "Bumping $CURRENT -> $NEW ($BUMP)"

# --- Check for clean working tree ---
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: working tree is dirty. Commit or stash changes first."
  exit 1
fi

# --- Update version in plugin.json ---
sed -i.bak "s/\"version\": \"$CURRENT\"/\"version\": \"$NEW\"/" "$PLUGIN_JSON"
rm -f "${PLUGIN_JSON}.bak"

# --- Prepend new section to CHANGELOG ---
DATE=$(date +%Y-%m-%d)
HEADER="## ${TAG} (${DATE})"

# Check if there's already an entry for this tag
if grep -q "^## ${TAG}" "$CHANGELOG"; then
  echo "CHANGELOG already has entry for ${TAG}, skipping."
else
  TMPFILE=$(mktemp)
  {
    echo "# Changelog"
    echo ""
    echo "$HEADER"
    echo ""
    echo "- "
    echo ""
    # Skip the first line (# Changelog) and blank line
    tail -n +2 "$CHANGELOG"
  } > "$TMPFILE"
  mv "$TMPFILE" "$CHANGELOG"

  # Open editor for changelog entry
  if [ -n "${EDITOR:-}" ]; then
    "$EDITOR" "$CHANGELOG"
  elif command -v code &> /dev/null; then
    code --wait "$CHANGELOG"
  elif command -v vim &> /dev/null; then
    vim "$CHANGELOG"
  else
    echo ""
    echo "Edit $CHANGELOG to fill in release notes, then press Enter."
    read -r
  fi
fi

# --- Commit, tag, push ---
git add "$PLUGIN_JSON" "$CHANGELOG"
git commit -m "release: ${TAG}"
git tag -a "$TAG" -m "Release ${TAG}"
git push origin main
git push origin "$TAG"

echo ""
echo "Released ${TAG}"
echo "  GitHub Actions will build and publish binaries."
echo "  https://github.com/SimoneLocatelli/claudewatch/releases/tag/${TAG}"

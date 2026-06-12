#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BRANCH_NAME="${1:-deploy}"
REMOTE_NAME="${2:-origin}"
STAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
AUTHOR_NAME="$(git -C "$ROOT_DIR" config user.name || true)"
AUTHOR_EMAIL="$(git -C "$ROOT_DIR" config user.email || true)"

REMOTE_URL="$(git -C "$ROOT_DIR" remote get-url "$REMOTE_NAME")"
TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

BUNDLE_DIR="$TMP_DIR/bundle"
REPO_DIR="$TMP_DIR/repo"

"$ROOT_DIR/scripts/build-production-bundle.sh" "$BUNDLE_DIR"

mkdir -p "$REPO_DIR"
cp -R "$BUNDLE_DIR"/. "$REPO_DIR"/

git -C "$REPO_DIR" init -q
git -C "$REPO_DIR" checkout -q -b "$BRANCH_NAME"
git -C "$REPO_DIR" config user.name "${AUTHOR_NAME:-idol-auth deploy}"
git -C "$REPO_DIR" config user.email "${AUTHOR_EMAIL:-deploy@local}"
git -C "$REPO_DIR" add .
git -C "$REPO_DIR" commit -q -m "chore: publish production bundle $STAMP"
git -C "$REPO_DIR" remote add "$REMOTE_NAME" "$REMOTE_URL"
git -C "$REPO_DIR" push --force "$REMOTE_NAME" "$BRANCH_NAME:$BRANCH_NAME"

echo "pushed slim deployment branch: $REMOTE_NAME/$BRANCH_NAME"

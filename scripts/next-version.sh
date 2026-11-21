#!/usr/bin/env bash
#
# Compute the next semver tag from Conventional Commits since the last release.
#
# Bump table (precedence: breaking > feat > fix/perf; multiple commits pick the
# highest-ranked one):
#   fix, perf                                  -> patch
#   feat                                       -> minor
#   `!` before the `:` in the subject          -> breaking
#   footer line `BREAKING CHANGE:` / `BREAKING-CHANGE:` -> breaking
# All other types (docs, chore, test, ci, build, style, refactor, revert) and
# non-conventional subjects produce no release.
#
# Breaking bumps: major > 0 -> major+1, minor/patch reset; major == 0 ->
# minor+1, patch reset (0.x stays 0.x).
#
# Base is the highest stable vX.Y.Z tag reachable from HEAD (pre-releases are
# ignored); with no such tag the base is v0.0.0 and the whole history is
# scanned. Prints only the next tag (e.g. `v0.3.0`), or nothing when there is
# nothing to release. Types are matched against subjects only; breaking-change
# footers against full commit messages.

set -euo pipefail

tags=$(git tag --merged HEAD --list 'v*' --sort=-version:refname)

base=''
while IFS= read -r tag; do
  if [[ $tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    base=$tag
    break
  fi
done <<<"$tags"

if [[ -n $base ]]; then
  range="$base..HEAD"
else
  base=v0.0.0
  range=HEAD
fi

subjects=$(git log --format=%s "$range")
bodies=$(git log --format=%B "$range")

if grep -qE '^[A-Za-z][A-Za-z0-9_-]*(\([^)]*\))?!:' <<<"$subjects" ||
  grep -qE '^BREAKING[ -]CHANGE:' <<<"$bodies"; then
  bump=breaking
elif grep -qE '^feat(\([^)]*\))?!?:' <<<"$subjects"; then
  bump=feat
elif grep -qE '^(fix|perf)(\([^)]*\))?!?:' <<<"$subjects"; then
  bump=fix
else
  exit 0
fi

IFS=. read -r major minor patch <<<"${base#v}"
major=$((10#$major))
minor=$((10#$minor))
patch=$((10#$patch))

case $bump in
breaking)
  if ((major > 0)); then
    major=$((major + 1))
    minor=0
  else
    minor=$((minor + 1))
  fi
  patch=0
  ;;
feat)
  minor=$((minor + 1))
  patch=0
  ;;
fix)
  patch=$((patch + 1))
  ;;
esac

printf 'v%s.%s.%s\n' "$major" "$minor" "$patch"

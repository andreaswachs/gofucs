#!/usr/bin/env bash
#
# Test harness for next-version.sh. Builds throwaway git repos under $TMPDIR
# and asserts the computed version for each scenario.

set -euo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
next_version="$script_dir/next-version.sh"

tmp_root=$(mktemp -d)
trap 'rm -rf "$tmp_root"' EXIT

repo=''
case_count=0

# new_repo <name>: fresh repo with identity configured.
new_repo() {
  repo="$tmp_root/$1"
  mkdir -p "$repo"
  git -C "$repo" init -q
  git -C "$repo" config user.name t
  git -C "$repo" config user.email t@t
}

# commit <subject> [body]
commit() {
  local subject=$1 body=${2-}
  if [[ -n $body ]]; then
    git -C "$repo" commit -q --allow-empty -m "$subject" -m "$body"
  else
    git -C "$repo" commit -q --allow-empty -m "$subject"
  fi
}

tag() { git -C "$repo" tag "$1"; }

# check <case name> <expected stdout>
check() {
  local name=$1 expected=$2 actual
  actual=$(cd "$repo" && bash "$next_version")
  case_count=$((case_count + 1))
  printf 'case %d: %s\n' "$case_count" "$name"
  if [[ $actual != "$expected" ]]; then
    printf 'FAIL: %s\n  expected: %q\n  actual:   %q\n' "$name" "$expected" "$actual" >&2
    return 1
  fi
}

# --- no tag: nothing release-worthy -----------------------------------------
new_repo no-tag-chore
commit 'chore: initial'
check 'no tag + chore only' ''

# --- no tag: feat -> v0.1.0 ------------------------------------------------
new_repo no-tag-feat
commit 'feat(fucs): create poc'
check 'no tag + feat' 'v0.1.0'

# --- fixes/perf on top of a tag --------------------------------------------
new_repo fix
commit 'feat: base'
tag v0.1.0
commit 'fix: correct parsing'
check 'v0.1.0 + fix' 'v0.1.1'

new_repo perf
commit 'feat: base'
tag v0.1.0
commit 'perf: faster loop'
check 'v0.1.0 + perf' 'v0.1.1'

# --- feat -> minor ---------------------------------------------------------
new_repo feat
commit 'feat: base'
tag v0.1.0
commit 'feat: new api'
check 'v0.1.0 + feat' 'v0.2.0'

# --- breaking, 0.x stays 0.x ----------------------------------------------
new_repo bang-zero
commit 'feat: base'
tag v0.1.0
commit 'feat!: drop old api'
check 'v0.1.0 + feat!' 'v0.2.0'

# --- breaking, major > 0 ---------------------------------------------------
new_repo bang-major
commit 'feat: base'
tag v1.2.3
commit 'fix(api)!: drop old api'
check 'v1.2.3 + fix(api)!' 'v2.0.0'

# --- breaking footer -------------------------------------------------------
new_repo breaking-footer
commit 'feat: base'
tag v1.2.3
commit 'fix: adjust api' 'BREAKING CHANGE: drop X'
check 'v1.2.3 + BREAKING CHANGE footer' 'v2.0.0'

# --- non-release-worthy commits only --------------------------------------
new_repo docs-only
commit 'feat: base'
tag v1.2.3
commit 'docs: tweak readme'
commit 'test: add case'
commit 'chore: bump deps'
check 'v1.2.3 + docs/test/chore only' ''

# --- HEAD exactly at the tag ----------------------------------------------
new_repo head-at-tag
commit 'feat: base'
tag v1.2.3
check 'HEAD exactly at v1.2.3' ''

# --- highest reachable tag wins -------------------------------------------
new_repo highest-tag
commit 'feat: first'
tag v0.1.0
commit 'feat: second'
tag v0.2.0
commit 'feat: third'
check 'highest of v0.1.0/v0.2.0 + feat' 'v0.3.0'

# --- breaking wins over feat/fix ------------------------------------------
new_repo precedence
commit 'feat: base'
tag v1.2.3
commit 'fix: one'
commit 'feat: two'
commit 'chore: three' 'BREAKING CHANGE: drop X'
check 'breaking footer + fix + feat' 'v2.0.0'

# --- `!` after the colon is not breaking ----------------------------------
new_repo bang-after-colon
commit 'feat: base'
tag v1.2.3
commit 'fix: wow!'
check 'v1.2.3 + fix with ! after colon' 'v1.2.4'

# --- non-conventional subject with ! before a colon is not breaking --------
new_repo bang-nonconventional
commit 'feat: base'
tag v1.2.3
commit 'WIP! this: not conventional'
check 'non-conventional ! before colon' ''

# --- types are matched on the subject only --------------------------------
new_repo body-only-type
commit 'feat: base'
tag v1.2.3
commit 'update stuff' 'feat: x'
check 'non-conventional subject with feat in body' ''

printf 'PASS: %d cases\n' "$case_count"

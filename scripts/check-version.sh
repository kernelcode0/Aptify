#!/usr/bin/env bash
set -euo pipefail

version="$(tr -d '[:space:]' < VERSION)"
numeric="${version#v}"

if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "VERSION must use vMAJOR.MINOR.PATCH; got: $version" >&2
  exit 1
fi

check() {
  local file="$1"
  local expected="$2"
  if ! grep -Fq -- "$expected" "$file"; then
    echo "$file is not synchronized with VERSION ($version)" >&2
    exit 1
  fi
}

check cmd/server/main.go "var version = \"$version\""
check cmd/aptify-cli/main.go "var version = \"$version\""
check Dockerfile "ARG VERSION=$version"
check docker-compose.yml "VERSION:-$version"
check README.md "version-$version-blue.svg"
check .github/ISSUE_TEMPLATE/bug_report.md "$version or commit hash"
check internal/api/security_test.go "\"$version\""

node - "$numeric" <<'NODE'
const fs = require('fs')
const expected = process.argv[2]
const pkg = JSON.parse(fs.readFileSync('web/package.json', 'utf8'))
const lock = JSON.parse(fs.readFileSync('web/package-lock.json', 'utf8'))
if (pkg.version !== expected || lock.version !== expected || lock.packages?.['']?.version !== expected) {
  console.error(`Frontend package metadata is not synchronized with VERSION (v${expected})`)
  process.exit(1)
}
NODE

echo "Version metadata is synchronized at $version."

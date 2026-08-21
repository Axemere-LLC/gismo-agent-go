#!/usr/bin/env bash
set -uo pipefail

# Table-driven test for bump-version.sh's next_version arithmetic, plus one
# case exercising rewrite_version_go against a temp copy of the constant
# file. Sources the script (which no-ops its main() when sourced) rather
# than shelling out, so agent/server.go itself is never touched.
#
# Usage: bump-version.test.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=bump-version.sh
source "$SCRIPT_DIR/bump-version.sh"

fail=0

# $1 = current, $2 = level, $3 = expected next version (or "ERROR")
check() {
  local current="$1" level="$2" expected="$3"
  local got
  if got="$(next_version "$current" "$level" 2>/dev/null)"; then
    if [ "$expected" = "ERROR" ]; then
      echo "FAIL next_version($current, $level) = '$got', expected an error"
      fail=1
    elif [ "$got" != "$expected" ]; then
      echo "FAIL next_version($current, $level) = '$got', expected '$expected'"
      fail=1
    else
      echo "OK   next_version($current, $level) = $got"
    fi
  else
    if [ "$expected" = "ERROR" ]; then
      echo "OK   next_version($current, $level) correctly errored"
    else
      echo "FAIL next_version($current, $level) errored, expected '$expected'"
      fail=1
    fi
  fi
}

# Field carries across double digits — a naive string-increment would break
# 0.9.0 -> 1.0.0 instead of 0.10.0.
check "0.9.0" "minor" "0.10.0"
check "0.1.9" "patch" "0.1.10"
# minor resets patch; major resets both.
check "1.2.3" "minor" "1.3.0"
check "1.2.3" "major" "2.0.0"
check "0.1.0" "patch" "0.1.1"
check "1.2.3" "patch" "1.2.4"
# Malformed input and unknown level both reject rather than guessing.
check "1.2" "patch" "ERROR"
check "1.2.3" "revision" "ERROR"
check "v1.2.3" "patch" "ERROR"

# rewrite_version_go: run against a temp copy, never the real file.
tmpfile="$(mktemp)"
trap 'rm -f "$tmpfile" "${tmpfile}.bak"' EXIT
cat > "$tmpfile" <<'EOF'
package agent

const Name = "gismo-agent-go"

const Version = "0.1.0"
EOF
rewrite_version_go "$tmpfile" "0.2.0"
if grep -q '^const Version = "0.2.0"$' "$tmpfile"; then
  echo "OK   rewrite_version_go wrote const Version = \"0.2.0\""
else
  echo "FAIL rewrite_version_go did not update the Version constant as expected"
  fail=1
fi
if grep -q '^const Name = "gismo-agent-go"$' "$tmpfile"; then
  echo "OK   rewrite_version_go left Name untouched"
else
  echo "FAIL rewrite_version_go modified an unrelated line"
  fail=1
fi

echo
if [ "$fail" -ne 0 ]; then
  echo "bump-version.test.sh: FAILED"
  exit 1
fi

echo "bump-version.test.sh: all cases pass"

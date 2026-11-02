#!/usr/bin/env bash
#
# Warns about TODO and FIXME comments that don't link to a GitHub issue.
#
# Every such comment must have the form:
#
#   TODO(#123): Describe what is left to do.
#
# i.e., the keyword in upper case, the issue number in parentheses, a colon, a
# space, and a non-empty description. Only files tracked by git are checked.
#
# Usage: tool/lint-todos.sh [--strict]
#
# Warnings are printed as GitHub Actions annotations when run in GitHub
# Actions, and as <path>:<line>: <message> otherwise. The exit status is 0
# even if there are warnings, unless --strict is given, in which case it is 1.

set -euo pipefail

strict=false
case "${1:-}" in
  "") ;;
  --strict) strict=true ;;
  *) echo "Usage: tool/lint-todos.sh [--strict]" >&2; exit 1 ;;
esac

cd "$(git rev-parse --show-toplevel)"

# The keyword must follow a comment marker so that prose and string literals
# that merely mention it are ignored.
readonly KEYWORD_RE='(//|#|--|/\*|\*|<!--)[[:space:]]*(TODO|FIXME)([^A-Za-z0-9_]|$)'
readonly VALID_RE='(TODO|FIXME)\(#[0-9]+\): [^ ]'

count=0
# Test data and the script itself contain these keywords as plain text.
while IFS=: read -r path line text; do
  if [[ "$text" =~ $VALID_RE ]]; then
    continue
  fi
  # Trims the indentation so that the comment itself is the message.
  message="${text#"${text%%[![:space:]]*}"}"
  if [[ -n "${GITHUB_ACTIONS:-}" ]]; then
    echo "::warning file=$path,line=$line::${message//%/%25}"
  else
    echo "$path:$line: $message"
  fi
  count=$((count + 1))
done < <(git grep -niIE "$KEYWORD_RE" -- . \
  ':!server/itest/testdata' ':!tool/lint-todos.sh' || true)

if [[ $count -gt 0 ]]; then
  echo "$count malformed TODO comment(s) found. Use 'TODO(#<issue number>): <description>'." >&2
  [[ "$strict" == true ]] && exit 1
fi
exit 0

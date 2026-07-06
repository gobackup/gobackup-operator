#!/usr/bin/env bash
# check-apiversion.sh - guard against reintroducing the pre-v1alpha1 API version.
# Fails if any tracked file references the bare `gobackup.io/v1` group/version
# (the project moved to `gobackup.io/v1alpha1`). Docs are excluded.
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

# Match `gobackup.io/v1` NOT followed by `alpha` (so /v1alpha1 is allowed).
if git grep -nE 'gobackup\.io/v1([^a]|$)' -- ':!*.md'; then
  echo "ERROR: found bare 'gobackup.io/v1' references above. Use 'gobackup.io/v1alpha1'." >&2
  exit 1
fi

echo "OK: no bare gobackup.io/v1 references."

#!/usr/bin/env bash
set -euo pipefail
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
"$REPO_ROOT/scripts/bootstrap-testsite.sh"
exec dotnet run --project "$REPO_ROOT/testsite"

#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TEMPLATE_DIR="$REPO_ROOT/testsite-template"
TARGET_DIR="$REPO_ROOT/testsite"
FORCE=0

for arg in "$@"; do
  case "$arg" in
    --force) FORCE=1 ;;
    *) echo "Unknown argument: $arg" >&2; exit 1 ;;
  esac
done

if [[ -d "$TARGET_DIR" && "$FORCE" -eq 0 ]]; then
  echo "testsite/ already exists — skipping bootstrap (use --force to recreate)."
  exit 0
fi

if [[ -d "$TARGET_DIR" && "$FORCE" -eq 1 ]]; then
  echo "Removing existing testsite/ (--force)..."
  rm -rf "$TARGET_DIR"
fi

echo "Bootstrapping testsite/ from testsite-template/..."
mkdir -p "$TARGET_DIR"
rsync -a \
  --exclude 'bin/' \
  --exclude 'obj/' \
  --exclude 'umbraco/' \
  --exclude 'wwwroot/' \
  --exclude 'appsettings.local.json' \
  --exclude '.gitignore' \
  "$TEMPLATE_DIR"/ "$TARGET_DIR"/

mv "$TARGET_DIR/testsite-template.csproj" "$TARGET_DIR/testsite.csproj"

cat > "$TARGET_DIR/appsettings.local.json" <<'JSON'
{
  "ConnectionStrings": {
    "umbracoDbDSN": "Data Source=|DataDirectory|/Umbraco.sqlite.db;Cache=Shared;Foreign Keys=True;Pooling=True",
    "umbracoDbDSN_ProviderName": "Microsoft.Data.Sqlite"
  }
}
JSON

echo "testsite/ ready. Run: dotnet run --project testsite"

---
name: umbraco-cli
description: >
  Use when managing an Umbraco CMS instance via the umbraco CLI tool.
  Covers content, media, users, schema, and system operations against
  the Umbraco Management API (v14+). Compatible with OpenCode and Claude Code.
---

# Umbraco Management CLI — Agent Skill

## Overview

`umbraco` is a CLI tool for managing Umbraco CMS instances via the Management API (Umbraco 14+). It is designed for both human operators and AI agents.

## Prerequisites

- The CLI binary (`umbraco`) must be on `$PATH`.
- A credential profile must be configured (see **Authentication** below).
- The target Umbraco instance must have an **API User** created in the backoffice with Client Credentials flow.

---

## Authentication

### One-time setup

Run the interactive auth command to add a named profile:

```sh
umbraco auth add <profile-name>
# Prompts: Base URL, Client ID, Client Secret
# Validates credentials immediately by fetching a token.
```

Credentials are stored in `~/.config/umbraco-cli/config.toml`.

### Managing profiles

```sh
umbraco auth list                  # List all profiles (* marks default)
umbraco auth set-default <name>    # Change the default profile
umbraco auth remove <name>         # Delete a profile
```

### Selecting a profile at runtime

Priority order: `--profile` flag > `UMBRACO_PROFILE` env var > `default_profile` in config.

```sh
umbraco --profile staging content list
UMBRACO_PROFILE=production umbraco system info
```

---

## Global Flags

These flags apply to every command:

| Flag | Description |
|---|---|
| `--profile <name>` | Named credential profile to use |
| `--json` | Output as JSON (machine-readable) |
| `--base-url <url>` | Override the base URL for this invocation |

**Always use `--json` in agent workflows** for reliable parsing.

---

## Commands

All commands below are verified against Umbraco 17 (Management API v1).

### Content (Documents)

```sh
# List root-level documents (paginated) — uses GET /tree/document/root
umbraco content list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, HAS CHILDREN, IS FOLDER
# JSON shape: {"total": N, "items": [{"id":"...","name":"...","hasChildren":false,"isFolder":false,...}]}

# Get a document by UUID — returns full document JSON
umbraco content get <id> [--json]

# Create a document from a JSON file
umbraco content create --file document.json [--json]

# Create a document from stdin
echo '{"contentTypeAlias":"blogPost","values":[...]}' | umbraco content create --file - [--json]

# Update a document
umbraco content update <id> --file updated.json [--json]

# Delete a document
umbraco content delete <id>

# Publish a document (all cultures)
umbraco content publish <id>

# Publish specific cultures
umbraco content publish <id> --cultures en-US,da-DK

# Unpublish a document
umbraco content unpublish <id> [--cultures en-US]
```

### Media

```sh
# List root-level media items (paginated) — uses GET /tree/media/root
umbraco media list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, HAS CHILDREN, IS FOLDER
# JSON shape: {"total": N, "items": [{"id":"...","name":"...","hasChildren":false,"isFolder":false,...}]}

# Get a media item by ID — returns full media JSON
umbraco media get <id> [--json]

# Upload a file as a media item (two-step: temp upload → create)
umbraco media upload ./photo.jpg [--parent <folder-id>] [--media-type Image]

# Delete a media item
umbraco media delete <id>
```

### Users

```sh
# List backoffice users — uses GET /user
umbraco user list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, EMAIL, STATE

# Get a user by ID
umbraco user get <id> [--json]

# List user groups — uses GET /user-group
umbraco user group-list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, ALIAS, DESCRIPTION
```

### Schema

```sh
# List document types — uses GET /tree/document-type/root
umbraco schema doctype list [--skip N] [--take N] [--json]
# Table columns: ID, NAME

# Get a document type by ID
umbraco schema doctype get <id> [--json]

# Create a document type from a JSON file
umbraco schema doctype create --file doctype.json [--json]
# Prints: "Document type created: <id>"
# JSON output: {"id": "<uuid>"}

# Update a document type
umbraco schema doctype update <id> --file doctype.json [--json]

# Delete a document type
umbraco schema doctype delete <id>

# List data types — uses GET /filter/data-type
umbraco schema datatype list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, EDITOR ALIAS
# JSON shape: {"total":37,"items":[{"id":"...","name":"...","editorAlias":"Umbraco.TextBox",...}]}

# Get a data type by ID
umbraco schema datatype get <id> [--json]

# Create a data type from a JSON file
umbraco schema datatype create --file datatype.json [--json]
# Prints: "Data type created: <id>"
# JSON output: {"id": "<uuid>"}

# Update a data type
umbraco schema datatype update <id> --file datatype.json [--json]

# Delete a data type
umbraco schema datatype delete <id>
```

#### Minimum viable JSON bodies

**Document type create/update:**
```json
{
  "name": "Blog Post",
  "alias": "blogPost",
  "icon": "icon-document",
  "allowedAsRoot": true,
  "variesByCulture": false,
  "variesBySegment": false,
  "isElement": false,
  "properties": [],
  "containers": [],
  "compositions": [],
  "allowedTemplates": [],
  "defaultTemplate": null,
  "allowedDocumentTypes": [],
  "cleanup": {"preventCleanup": false}
}
```

Properties inside a document type require:
- `id` — client-generated UUID
- `alias` — camelCase string
- `name` — display name
- `dataType` — `{"id": "<datatype-uuid>"}`
- `sortOrder` — integer
- `validation` — `{"mandatory": false, "mandatoryMessage": null, "regEx": null, "regExMessage": null}`
- `appearance` — `{"labelOnTop": false}`
- `variesByCulture` — bool
- `variesBySegment` — bool

**Data type create/update:**
```json
{
  "name": "My Textbox",
  "editorAlias": "Umbraco.TextBox",
  "editorUiAlias": "Umb.PropertyEditorUi.TextBox",
  "values": []
}
```

### System

```sh
# Print server / version info — uses GET /server/information
umbraco system info [--json]
# Example output:
#   assemblyVersion   17.3.3
#   baseUtcOffset     01:00:00
#   runtimeMode       BackofficeDevelopment
#   version           17.3.3+ade77ad

# List health check groups — uses GET /health-check-group
umbraco system health [--json]
# Table column: GROUP (name only — the API does not return an id field)
# Example groups: Configuration, Data Integrity, Live Environment, Permissions, Security, Services

# Trigger a published cache rebuild — uses POST /published-cache/rebuild
umbraco system cache-rebuild
```

---

## Agent Patterns

### Chaining: create schema → create content

```sh
# 1. Create a data type
umbraco schema datatype create --file - <<'EOF' --json
{"name":"My Textbox","editorAlias":"Umbraco.TextBox","editorUiAlias":"Umb.PropertyEditorUi.TextBox","values":[]}
EOF
# Returns: {"id": "<datatype-uuid>"}

# 2. Create a document type referencing that data type
umbraco schema doctype create --file doctype.json --json
# Returns: {"id": "<doctype-uuid>"}

# 3. Create a content node using the new document type
umbraco content create --file - <<'EOF' --json
{
  "documentType": {"id": "<doctype-uuid>"},
  "template": null,
  "values": [{"alias":"title","value":"Hello"}],
  "variants": [{"culture": null, "segment": null, "name": "My Page"}]
}
EOF

# 4. Publish it
umbraco content publish <id-from-step-3>
```

### Chaining: get schema → create content

```sh
# 1. Find the document type name/ID from the tree
umbraco schema doctype list --json | jq '.items[] | select(.name=="Blog Post")'

# 2. Get the full document type to find its alias
umbraco schema doctype get <id-from-step-1> --json | jq '{alias, name}'

# 3. Create a document using that alias
umbraco content create --file - <<'EOF'
{
  "contentTypeAlias": "blogPost",
  "parentId": null,
  "values": [
    { "alias": "title",   "value": "Hello World" },
    { "alias": "bodyText","value": "<p>Content here</p>" }
  ]
}
EOF

# 4. Publish it
umbraco content publish <id-from-step-3>
```

### Pagination: iterate all documents

```sh
# Get all documents in pages of 50
umbraco content list --skip 0  --take 50 --json
umbraco content list --skip 50 --take 50 --json
# Continue until `total` is reached.
```

### Multi-environment workflow

```sh
# Export from staging, import to production
umbraco --profile staging  content get <id> --json > doc.json
umbraco --profile production content create --file doc.json --json
```

---

## Output Format

Without `--json`: human-readable tables printed to stdout.

With `--json`: JSON object/array to stdout, errors to stderr as `{"error": "..."}`.

Exit code `0` = success, non-zero = failure.

---

## Config File Reference

Location: `~/.config/umbraco-cli/config.toml`

```toml
default_profile = "production"

[profiles.production]
base_url      = "https://mysite.com"
client_id     = "umbraco-back-office-my-client"
client_secret = "s3cr3t"

[profiles.staging]
base_url      = "https://staging.mysite.com"
client_id     = "umbraco-back-office-staging"
client_secret = "s3cr3t2"
```

Environment variables take precedence over config file values. Prefix: `UMBRACO_`.

---

## Creating an API User (one-time backoffice task)

1. Log into the Umbraco backoffice.
2. Go to **Users** → **API Users** → **Create**.
3. Give it a name and assign appropriate user groups.
4. Copy the generated **Client ID** and **Client Secret**.
5. Run `umbraco auth add <profile-name>` and supply those values.

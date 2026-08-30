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

All commands below are verified against Management API v1 (originally checked against
Umbraco 17; the newer commands below were exercised live against Umbraco 18.1.0 via this
project's own `testsite-template/` — see `CLAUDE.md`).

### Content (Documents)

```sh
# List root-level documents (paginated) — uses GET /tree/document/root
umbraco content list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, HAS CHILDREN, IS FOLDER
# JSON shape: {"total": N, "items": [{"id":"...","name":"...","hasChildren":false,"isFolder":false,...}]}

# List the children of a specific document — uses GET /tree/document/children
umbraco content list --parent <id> [--skip N] [--take N] [--json]
# Same shape as above. This is the only way to browse below the root — use it
# to explore an existing site's tree, or to find a parent to hang test content under.

# Get a document by UUID — returns full document JSON
umbraco content get <id> [--json]

# Validate a document body without persisting it — uses POST /document/validate
umbraco content validate --file document.json [--json]
# Same body shape as `content create`. Verified against a live instance: it does
# catch structural errors (e.g. a nonexistent documentType id → 404), but it did
# NOT catch an unknown property alias or an empty variants array in testing —
# treat a "Document is valid" result as necessary, not sufficient; `content create`
# can still fail afterwards on issues this endpoint doesn't check.

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

# Create a document and publish it in one call — uses POST /document/create-and-publish
umbraco content create-and-publish --file document.json [--cultures en-US,da-DK] [--json]
# Same body shape as `content create`. A client-generated id is added to the
# body automatically so this always reports back the new document's ID.

# Search documents by name — uses GET /item/document/search (always returns JSON)
umbraco content search "<query>" [--skip N] [--take N] [--trashed] [--culture en-US] [--parent <id>]

# List what references a document (e.g. via a content picker) — always returns JSON
umbraco content referenced-by <id> [--skip N] [--take N]
# Check this before deleting, to avoid leaving dangling references elsewhere.

# Bulk-check whether a set of documents are referenced — always returns JSON
umbraco content are-referenced --id <id> [--id <id> ...] [--skip N] [--take N]

# Copy a document to a new parent (omit --target to copy to root)
umbraco content copy <id> --target <parent-id> [--relate-to-original] [--include-descendants=false]
# The API does not reliably return the new document's ID — if this command
# can't recover one, list the target parent's children to find the copy.

# Move a document to a new parent (omit --target to move to root)
umbraco content move <id> --target <parent-id>

# Move a document to the recycle bin (reversible, unlike 'content delete')
umbraco content trash <id>

# Restore a document from the recycle bin (omit --target to restore to its original parent)
umbraco content restore <id> [--target <parent-id>]

# Browse the recycle bin
umbraco content trash-list [--parent <id>] [--skip N] [--take N] [--json]

# Permanently delete one item already in the recycle bin
umbraco content trash-delete <id>

# Permanently empty the whole recycle bin
umbraco content trash-empty

# Auto-sort a document's children (or the root's) by name or date
umbraco content sort-children [parent-id] [--field Name|CreateDate|UpdateDate] [--direction Ascending|Descending]
```

#### Content Blueprints

A blueprint is a saved content starting point — the fastest way to spin up
repeatable test content.

```sh
# List blueprints — uses GET /tree/document-blueprint/root (or /children with --parent)
umbraco content blueprint list [--parent <id>] [--skip N] [--take N] [--json]

# Get a blueprint by ID — always returns JSON
umbraco content blueprint get <id>

# Get a pre-filled 'content create' body from a blueprint — always returns JSON
umbraco content blueprint scaffold <id>
# Edit the name/values and pipe straight into: umbraco content create --file -

# Create a blueprint directly from a JSON file (same shape as 'content create')
umbraco content blueprint create --file blueprint.json [--json]

# Create a blueprint FROM an existing document — the recommended path: pick a
# "golden" example node and turn it into a reusable starting point
umbraco content blueprint create-from <content-id> --name "Blog Post Starter" [--parent <blueprint-folder-id>] [--json]

# Delete a blueprint
umbraco content blueprint delete <id>
```

### Media

```sh
# List root-level media items (paginated) — uses GET /tree/media/root
umbraco media list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, HAS CHILDREN, IS FOLDER
# JSON shape: {"total": N, "items": [{"id":"...","name":"...","hasChildren":false,"isFolder":false,...}]}

# List the contents of a specific media folder — uses GET /tree/media/children
umbraco media list --parent <folder-id> [--skip N] [--take N] [--json]

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

# Search document types by name — uses GET /item/document-type/search
umbraco schema doctype search "<query>" [--skip N] [--take N] [--json]
# Use this to find a doctype's ID/alias without paging through the whole tree.

# List document types allowed as children of a document type
umbraco schema doctype allowed-children <id> [--skip N] [--take N] [--json]

# List document type IDs allowed as a parent of a document type (counterpart of allowed-children)
umbraco schema doctype allowed-parents <id> [--json]

# List document types allowed at the content root
umbraco schema doctype allowed-at-root [--skip N] [--take N] [--json]

# List document types that use this document type as a composition (reverse lookup)
umbraco schema doctype composition-refs <id> [--json]

# List document types that could be composed INTO this document type
umbraco schema doctype available-compositions <id> [--json]
# Fetches the target doctype's current properties/compositions, then asks the
# API which other doctypes are compatible (no clashing property aliases, no cycles).
```

#### Patch-style editing (add/update/remove a single property, container, or composition)

The Management API only exposes a whole-document-type PUT — there is no
per-property PATCH. These commands do the fetch → modify → PUT round-trip for
you so you don't have to hand-edit the full JSON body just to add one field.

**Important, verified against a live instance:** Umbraco silently discards a container
that has zero properties when the document type is saved. That means a brand-new
container can only be created in the *same* PUT as the first property that uses it —
never as a standalone `container add` followed by a separate `property add --container
<id>`, because by the time the second command runs, the container from the first is
already gone. Use `property add`'s `--new-container-name`/`--new-container-type` flags
(below) to create the container and its first property atomically; only use
`container add` on its own to add an *additional* container once the document type
already has at least one property.

Also verified: reusing an **existing** container's exact name+type via
`--new-container-name` is rejected by this CLI on purpose — Umbraco matches/merges
containers by name+type (not id) on save, so a same-named "new" container silently
merges into the existing one under a fresh id, orphaning (and dropping) any property
still pointing at the old id. Use `--container <existing-id>` to attach to an existing
container instead.

```sh
# Add a property to an EXISTING container
umbraco schema doctype property add <doctype-id> \
  --alias myField --name "My Field" --data-type <datatype-id> \
  --container <container-id> \
  [--description "..."] [--sort-order N] \
  [--mandatory] [--mandatory-message "..."] [--reg-ex "..."] [--reg-ex-message "..."] \
  [--varies-by-culture] [--varies-by-segment] [--label-on-top]
# Fails if a property with that alias already exists.

# Add a property AND create its container in the same operation (the atomic,
# safe way to introduce a new tab/group — see warning above)
umbraco schema doctype property add <doctype-id> \
  --alias myField --name "My Field" --data-type <datatype-id> \
  --new-container-name "Content" --new-container-type Tab [--new-container-parent <tab-id>]
# Fails with a clear error if a container with that exact name+type already exists —
# use --container <existing-id> instead in that case.

# Update a property (only flags you pass are changed)
umbraco schema doctype property update <doctype-id> --alias myField [--name "..."] [--data-type <id>] ...

# Remove a property
umbraco schema doctype property remove <doctype-id> --alias myField

# Add an ADDITIONAL tab or group container (only safe once the document type already
# has ≥1 property — see warning above; otherwise it will silently not persist)
umbraco schema doctype container add <doctype-id> --name "Content" --type Tab [--parent <container-id>] [--sort-order N]
# --type must be exactly "Tab" or "Group" (a Group can be nested under a Tab via --parent)

# Update a container (only flags you pass are changed)
umbraco schema doctype container update <doctype-id> --id <container-id> [--name "..."] [--type Group] [--parent <id>] [--sort-order N]

# Remove a container
umbraco schema doctype container remove <doctype-id> --id <container-id>

# Add a composition (check available-compositions first)
umbraco schema doctype composition add <doctype-id> --composition-id <other-doctype-id> [--type Composition|Inheritance]
# --type defaults to "Composition"

# Remove a composition
umbraco schema doctype composition remove <doctype-id> --composition-id <other-doctype-id>
```

### Element Types

An element type is just a document type with `isElement: true` — there is no
separate Management API resource for it. These commands are thin wrappers
around `schema doctype` / `/document-type`, given their own command group
because they're the content model behind every Block List / Block Grid block.

```sh
# List element types — filters the doctype tree client-side (page-scoped; see 'search' for an exact match)
umbraco schema element list [--skip N] [--take N] [--json]

# Search element types by name — exact server-side isElement filter, prefer this over 'list'
umbraco schema element search "<query>" [--skip N] [--take N] [--json]

# Get an element type by ID (same resource as 'schema doctype get')
umbraco schema element get <id> [--json]

# Create an element type from a JSON file — same body as 'schema doctype create',
# but isElement/allowedAsRoot default to true/false when not set explicitly
umbraco schema element create --file element.json [--json]
# Then build it out with:
#   schema doctype container add <id> --name "Content" --type Tab
#   schema doctype property add <id> --alias ... --name ... --data-type ... --container <container-id>

# Update / delete (aliases for 'schema doctype update' / 'schema doctype delete')
umbraco schema element update <id> --file element.json [--json]
umbraco schema element delete <id>
```

Once an element type exists, wire it into a Block List / Block Grid data
type by creating (or updating) a data type with editor alias
`Umbraco.BlockList` or `Umbraco.BlockGrid` via `schema datatype create` /
`schema datatype update` — the block configuration (which element types are
allowed, per-block settings) lives in that data type's opaque `values` JSON,
which this CLI passes through as-is; there is no dedicated command for
editing it yet.

### Media Types

```sh
# List media types — uses GET /tree/media-type/root
umbraco schema mediatype list [--skip N] [--take N] [--json]

# Get a media type by ID
umbraco schema mediatype get <id> [--json]

# Create a media type from a JSON file (same shape as a document type, but no
# allowedTemplates/defaultTemplate, and "allowedMediaTypes" instead of "allowedDocumentTypes")
umbraco schema mediatype create --file mediatype.json [--json]

# Update a media type
umbraco schema mediatype update <id> --file mediatype.json [--json]

# Delete a media type
umbraco schema mediatype delete <id>
```

### Data Types

```sh
# List data types — uses GET /filter/data-type
umbraco schema datatype list [--skip N] [--take N] [--json]
# Table columns: ID, NAME, EDITOR ALIAS
# JSON shape: {"total":37,"items":[{"id":"...","name":"...","editorAlias":"Umbraco.TextBox",...}]}

# Filter by editor alias (client-side filter on the fetched page — pair with
# a larger --take if the match isn't on the default page)
umbraco schema datatype list --editor-alias Umbraco.TextBox [--json]
# Use this before creating a new data type, to check whether an existing site
# already wraps the editor you need.

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

# Search recent server log entries — uses GET /log-viewer/log
umbraco system logs [--level Warning,Error] [--search "<filter expression>"] \
  [--from 2026-08-01T00:00:00Z] [--to 2026-08-30T00:00:00Z] \
  [--order Ascending|Descending] [--skip N] [--take N] [--json]
# Useful for diagnosing an existing setup (startup errors, property editor
# exceptions) without shelling into the server's log files.

# Count log entries per level — uses GET /log-viewer/level-count
umbraco system log-levels [--from ...] [--to ...] [--json]
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

Verified against a live instance: whatever `clientId` you (or an API script) submit to
`POST /user/{id}/client-credentials` gets registered server-side with an
`umbraco-back-office-` prefix — the *usable* client ID for the token endpoint is
`umbraco-back-office-<what you submitted>`, not the literal value submitted. The
backoffice UI already shows you this final prefixed value to copy; a script calling
the API directly needs to prepend the prefix itself before calling `auth add`.

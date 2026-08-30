# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`umbraco-cli` is a Go CLI that wraps the Umbraco CMS Management API, built specifically to be a
lighter-weight, token-efficient alternative to an Umbraco MCP server for AI agents. The `skill/SKILL.md`
file embedded in the binary is the primary consumer-facing artifact: it teaches AI agents every
command, flag, output shape, and API endpoint the CLI exposes. When commands, flags, or output shapes
change, `skill/SKILL.md` must be updated to match — it is the contract agents rely on, not just
documentation.

## Build, run, test

```sh
go build -o umbraco .        # build the binary
go run . <args>               # run without building
go vet ./...                  # static checks
go test ./...                 # tests (none currently exist)
```

There is no Makefile, linter config, or CI pipeline in this repo — `go build`/`go vet` are the only
gates.

## Architecture

- **`main.go`** — embeds `skill/SKILL.md` via `//go:embed` and injects it into `cmd` package at
  startup (`cmd.SetSkillContent`). This is how `umbraco agent install` gets the skill content without
  reading from disk at runtime.
- **`cmd/`** — one file per Cobra command group (`content.go`, `content_lifecycle.go`,
  `content_blueprint.go`, `media.go`, `user.go`, `schema.go`, `schema_edit.go`, `schema_element.go`,
  `schema_mediatype.go`, `system.go`, `auth.go`, `agent.go`), plus `root.go` (global flags, profile
  resolution) and `helpers.go` (shared helpers like `readJSONInput`, `boolStr`, `newUUID`,
  `nullableString`). Every resource command file follows the same shape: a parent `*Cmd`
  cobra.Command, child subcommands (`list`/`get`/`create`/`update`/`delete`), each building an
  `*api.Client` via `newClientFromFlags()` and calling `Get`/`Post`/`Put`/`Delete`/`PostCreate` on it.
  `schema_edit.go` additionally does fetch→modify→PUT patch-style edits (see the container/property
  gotcha below) via its own `fetchDocTypeMap`/`putDocTypeMap` helpers.
- **`internal/api/client.go`** — thin HTTP client for the Management API (`/umbraco/management/api/v1`).
  Fetches an OAuth2 client-credentials Bearer token once at construction (`NewClient`). Notable quirk:
  Umbraco's create endpoints return `201` with an empty body — the new resource ID is read from the
  `umb-generated-resource` response header (falls back to `Location`), which is why creates use
  `PostCreate` instead of `Post`.
  - `--insecure` on a profile disables TLS verification via a custom `http.Transport` (intentional,
    for local dev against self-signed certs — flagged `nolint:gosec`).
- **`internal/config/config.go`** — Viper-backed TOML config at `~/.config/umbraco-cli/config.toml`,
  storing named `Profile`s (`base_url`, `client_id`, `client_secret`, `insecure`) plus a
  `default_profile`. Profile resolution order (see `activeProfile()` in `cmd/root.go`): `--profile`
  flag > `UMBRACO_PROFILE` env > `default_profile` in config.
- **`internal/output/output.go`** — all command output goes through this package (`output.JSON`,
  `output.Table`, `output.Line`, `output.Error`), gated by the global `--json` flag (`output.IsJSON()`).
  Never `fmt.Println` results directly in a command — route through here so `--json` works uniformly.
- **`cmd/agent.go`** — manages copying the embedded skill file into `~/.opencode/skill/umbraco-cli/`
  and `~/.claude/skills/umbraco-cli/` (`umbraco agent install/uninstall/status`). Install aborts if
  already installed; it does not overwrite silently.

## Testing against a real Umbraco instance

New commands must be exercised against a real running Umbraco instance before being considered
done — `go build`/`go vet`/`--help` are not sufficient. `testsite-template/` (tracked) is the
source of truth for a minimal Umbraco 18.1.0 + SQLite instance; `testsite/` (gitignored) is the
generated, running copy.

Prerequisites: `dotnet` SDK (net10.0), `python3` (stdlib only), `rsync`. No Docker/SQL Server
needed.

One-time setup:

```sh
scripts/run-testsite.sh                    # bootstraps testsite/ + dotnet run (first run installs
                                            # unattended via SQLite — the app's own console logging
                                            # does not reliably show through the piped stdout, so
                                            # don't wait on a specific log line; poll instead:
                                            # curl -sI http://localhost:58880/umbraco/login
python3 scripts/create-api-user.py         # provisions an API user + client-credentials (idempotent)
                                            # — prints the exact `umbraco auth add` command to run,
                                            # with the client id already correctly prefixed (see below)
```

Day to day: `scripts/run-testsite.sh` again (skips re-bootstrap if `testsite/` exists); Ctrl+C to
stop. To reset to a clean instance: `scripts/bootstrap-testsite.sh --force` (then re-run
`create-api-user.py`, since the database — and the API user in it — is wiped).

### Verified findings from live testing (2026-08-30, Umbraco 18.1.0)

The first real run against this instance surfaced several things that were previously only
inferred from reading the official MCP's generated API client, plus a couple of pre-existing bugs.
Recorded here so they aren't rediscovered from scratch:

- **A document-type container with zero properties is silently discarded by Umbraco on save.**
  `schema doctype container add` is therefore a no-op on its own for a brand-new container — the
  container must be created in the *same* PUT as its first property. `schema doctype property add`
  has `--new-container-name`/`--new-container-type` flags for exactly this (see `skill/SKILL.md`).
- **Reusing an existing container's exact name+type when creating a "new" one causes data loss.**
  Umbraco matches/merges containers by name+type, not id, so this silently orphans (drops) whatever
  property pointed at the old container's id. `property add` now refuses this case with an explicit
  error rather than risk it.
- **The API user's registered client ID gets an `umbraco-back-office-` prefix** applied server-side
  to whatever `clientId` is submitted to `POST /user/{id}/client-credentials` — the usable client ID
  for the token endpoint is `umbraco-back-office-<submitted-value>`, not the literal value. The
  backoffice UI already shows you this final value; `scripts/create-api-user.py` accounts for it.
- **The backoffice admin-cookie login sets a `Secure` cookie**, so the cookie-based part of API-user
  provisioning (login → PKCE authorize → code exchange) must run over HTTPS even though the testsite
  is otherwise HTTP-only — `client_credentials` calls (what the CLI itself actually uses) are
  unaffected since they don't use cookies. This is why `testsite-template/Properties/launchSettings.json`
  binds both an HTTP and an HTTPS port, and why `create-api-user.py` targets the HTTPS one with
  certificate verification disabled (the dev cert isn't in the system trust store here).
- **`content list`/`media list` had a real, pre-existing bug**: the tree endpoints have no top-level
  `"name"` field — it's nested in `variants[0].name` — so every row's name was silently blank. Fixed
  in `cmd/content.go`/`cmd/media.go` via a `resolveName()` post-processing step.
- **`content restore` 415'd when `--target` was omitted**: sending no request body at all (rather
  than an explicit `{"target": null}`) causes Umbraco's model binding to reject the request outright.
  Fixed by always sending an explicit body.
- **`content validate` is real but only partially strict**: it caught a nonexistent `documentType`
  id (404) but did *not* catch an unknown property alias or an empty `variants` array in testing.
  Treat a "Document is valid" result as necessary, not sufficient.

Everything else exercised worked as designed on the first try: `schema doctype`
search/allowed-children/allowed-parents/allowed-at-root/composition-refs/available-compositions,
`schema datatype list --editor-alias`, `content create-and-publish`, `content copy`, `content
trash`/`trash-list`/`restore`, `content search`/`referenced-by`/`are-referenced`, and `content
blueprint create-from`/`list`/`get`/`scaffold`.

## Conventions when adding a new command

- Follow the existing resource-file pattern (see `cmd/content.go`): define response structs with
  `json` tags matching the Management API shape, build list/get/create/update/delete as needed (not
  every resource needs all five — media/user/schema/system are asymmetric based on what the API
  supports).
- Use `newClientFromFlags()` to get an authenticated client; don't construct `api.Client` directly in
  commands.
- Use `readJSONInput(cmd, "file")` for commands that accept a JSON body via `--file` (support
  `--file=-` for stdin, matching the existing convention).
- Register new subcommands in each file's own `init()`, and add new top-level command groups to
  `rootCmd` in `cmd/root.go`'s `init()`.
- Update `skill/SKILL.md` alongside any command/flag/output change — it's the AI-agent-facing API
  reference and will silently drift otherwise.

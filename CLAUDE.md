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
- **`cmd/`** — one file per Cobra command group (`content.go`, `media.go`, `user.go`, `schema.go`,
  `system.go`, `auth.go`, `agent.go`), plus `root.go` (global flags, profile resolution) and
  `helpers.go` (shared helpers like `readJSONInput`, `boolStr`). Every resource command file follows
  the same shape: a parent `*Cmd` cobra.Command, child subcommands (`list`/`get`/`create`/`update`/
  `delete`), each building an `*api.Client` via `newClientFromFlags()` and calling `Get`/`Post`/`Put`/
  `Delete`/`PostCreate` on it.
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

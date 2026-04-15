# Umbraco CLI — Agent / Skill Installation

The file `skill/SKILL.md` in this repository is a self-contained agent skill that teaches AI coding assistants how to use the `umbraco` CLI. It works with both **OpenCode** and **Claude Code** (and any tool that supports the SKILL.md convention).

Once installed, the AI tool will automatically load the skill whenever it is relevant, giving it full knowledge of every `umbraco` command, flag, output shape, and agent pattern — no extra prompting required.

---

## Automatic installation (recommended)

If you have the `umbraco` binary on your `$PATH`, the easiest way is:

```sh
# Install for all supported tools at once
umbraco agent install

# Install for a specific tool only
umbraco agent install --tool opencode
umbraco agent install --tool claudecode

# Check installation status
umbraco agent status

# Remove
umbraco agent uninstall
umbraco agent uninstall --tool opencode
```

The command will **abort** if the skill is already installed, so it is safe to run more than once.

---

## Manual installation

### OpenCode

Copy `skill/SKILL.md` to the OpenCode global skill directory:

```sh
mkdir -p ~/.opencode/skill/umbraco-cli
cp skill/SKILL.md ~/.opencode/skill/umbraco-cli/SKILL.md
```

OpenCode picks up all `~/.opencode/skill/*/SKILL.md` files automatically. No config change needed.

**Verify:**

```sh
ls ~/.opencode/skill/umbraco-cli/SKILL.md
```

### Claude Code

Copy `skill/SKILL.md` to the Claude Code global skills directory:

```sh
mkdir -p ~/.claude/skills/umbraco-cli
cp skill/SKILL.md ~/.claude/skills/umbraco-cli/SKILL.md
```

Claude Code picks up all `~/.claude/skills/*/SKILL.md` files automatically. No config change needed.

**Verify:**

```sh
ls ~/.claude/skills/umbraco-cli/SKILL.md
```

---

## What the skill teaches the AI

- How to authenticate and manage named profiles (`umbraco auth`)
- All content, media, user, schema, and system commands with correct flags
- The `--json` flag and output shapes for reliable agent parsing
- Pagination patterns, multi-environment workflows, and chaining commands
- The exact Management API endpoint paths used under the hood (for debugging)

---

## Uninstalling manually

```sh
# OpenCode
rm -rf ~/.opencode/skill/umbraco-cli

# Claude Code
rm -rf ~/.claude/skills/umbraco-cli
```

---

## Keeping the skill up to date

The skill is embedded in the `umbraco` binary at build time, so running
`umbraco agent install` always installs the version that matches the binary you
have. After upgrading the binary, re-run:

```sh
umbraco agent uninstall && umbraco agent install
```

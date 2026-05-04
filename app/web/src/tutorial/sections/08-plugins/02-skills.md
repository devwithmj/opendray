# Skills

A *skill* is a markdown file describing a reusable prompt or
procedure. opendray syncs the configured skills directory into
each Claude session's `~/.claude/skills/` so Claude Code can
invoke them via slash commands.

## When to use

- "Run our standard PR review on this diff" — codify the
  reviewer's checklist as a skill, invoke with `/pr-review`.
- "Generate the bug-report template for issue X" — skill
  takes one arg, fills the template.
- Any prompt you find yourself typing more than twice — it's
  a skill candidate.

## File layout

```
~/.opendray/vault/skills/
  pr-review.md
  bug-report.md
  refactor-helper.md
```

Each `.md` file is one skill. The filename (sans `.md`) is the
slash-command name Claude registers. So `pr-review.md`
becomes `/pr-review` inside Claude.

## Skill markdown format

```markdown
# pr-review

Run a structured PR review.

## Args

- `$1` (optional) — branch to review against; defaults to `main`

## Prompt

You are reviewing a pull request against `$1`. Walk through the
diff one file at a time:

1. Identify intent — what is this PR trying to do?
2. Flag bugs, regressions, missed edge cases
3. Suggest tests to add
4. Comment on style only when it materially affects readability

Use `git diff` and `git log` to gather context. Output
should be a single comment block ready to paste into the PR
review.
```

The first H1 is the command name (canonicalised — must match the
filename for clarity, though opendray honours the H1 if they
diverge). Body sections are free-form; everything after the
`## Prompt` heading becomes the actual prompt sent to Claude.

`$1`, `$2`, … placeholders interpolate the slash command's
positional args. `$@` interpolates the full args string.

## Sync model

opendray watches the skills directory (default
`~/.opendray/vault/skills/`) and:

1. On opendray startup → reads every file → publishes a
   `skills.changed` event.
2. On any spawn of a Claude session → writes every skill into
   `~/.claude/skills/` (or the appropriate `CLAUDE_CONFIG_DIR`
   for the session's bound account).
3. On filesystem change (inotify on Linux, FSEvents on macOS)
   → re-publishes the changed event; running sessions pick up
   the new skills on their next prompt.

You can edit skills from the Notes editor (they're just markdown
in the vault), from Obsidian, from VS Code, or from any other
editor — opendray sees the changes either way.

## Disabling a skill

Two ways:

- **Soft disable**: prefix the filename with `_` (e.g.
  `_pr-review.md`). opendray skips files starting with `_`.
- **Hard disable**: delete the file. The skill goes away on
  next sync.

There's no per-session disable list at the moment (just
per-host).

## Naming rules

- Skill filenames must be `[a-z0-9-]+\.md` (lowercase, dashes).
- Names must not collide with built-in Claude slash commands
  (`/init`, `/clear`, etc.). opendray warns if you try.
- Keep skill bodies under ~5KB for fast Claude prompt loading.
- One skill per file — multi-skill files aren't parsed.

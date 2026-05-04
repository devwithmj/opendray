# Wiki links + backlinks

Wiki links are how Obsidian-style vaults stay connected. Type
`[[Other Note]]` in any note → it becomes a clickable link to
the matching file. Every note pointing at the current one shows
in the Backlinks pane.

## Creating a link

Inside the source-mode editor, type `[[`. opendray's caret
detection notices this and pops up a suggestion list of every
note title in the vault, fuzzy-matched against what you type
after the brackets.

![Wiki link suggestions popup](/tutorial/notes-wiki-link-suggest.png)

- **↑ / ↓** to highlight a candidate
- **Enter** or **Tab** to insert
- **Esc** to dismiss without inserting

The inserted text is `[[Note Title]]`. Saving the note triggers
a backlink-graph rebuild (incremental — only the affected files
re-scan).

## Resolution rules

When the renderer encounters `[[X]]`:

1. Look for a file `X.md` anywhere in the vault.
2. If multiple matches, prefer one in the same directory.
3. If still ambiguous, take the alphabetically-first.
4. If no match, the link renders as a "stub" with a dashed
   border — clicking it offers to create the note.

This matches Obsidian's behaviour, so notes round-trip cleanly
between the two.

## Backlinks pane

The right side of the note editor shows every other note that
contains `[[<this note's title>]]`. Click any backlink → opens
that note → its backlinks pane shows the chain.

Useful patterns:

- **Tag-style indices.** Create a note `Decisions Index.md` that
  every decision references. Every decision back-links here, so
  you have a one-stop view.
- **Project root.** Each project folder gets a `_README.md` that
  individual notes link to. The README's backlinks are the full
  project file list — ad hoc, but works.

## Aliases (alt text)

Wiki links support `|alias` for displayed text:

```
[[2026-Q2 Roadmap|the roadmap]]
```

Renders as "the roadmap" but still links to and back-links from
`2026-Q2 Roadmap.md`. opendray honours the alias for display
only — the backlink graph uses the canonical title.

## Limitations vs Obsidian

- **No nested links** (`[[X|[[Y]]]]`) — flatten them.
- **No section anchors yet** (`[[X#section]]`) — you can write
  them but they resolve to the file, not the heading.
- **No embed transclusion** (`![[X]]`) — opendray renders a
  link instead of embedding the body. Future feature.
- **Graph view** is not in opendray (only the Backlinks pane
  per note). For a global graph, open the vault in Obsidian
  itself.

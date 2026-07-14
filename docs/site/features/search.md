---
title: Search and Indexing
lang: en
translation_key: search
---

# Search and Indexing

Search is disabled by default. Enable the current SQLite search provider:

```yaml
search:
  enabled: true
  type: sqlite
```

After restarting go-drive, create an indexing job under **Admin → Other → File Index**. The `bleve` provider used in older versions is no longer supported.

## Initial indexing

Leave the path empty to index from the virtual root, or enter a subpath to index only that subtree. The job runs in the background; the page displays its status and progress and can abort an unfinished job.

Re-index affected paths when:

- Search is enabled for the first time.
- A Drive is added, deleted, or renamed.
- Files are changed directly outside go-drive.
- Index filtering rules change.
- A database is restored or storage is migrated.

Normal file operations performed through go-drive update the index, but changes in external systems cannot be detected automatically.

## Filtering rules

Each line begins with `+` or `-`, followed by a [path pattern](../reference/path-patterns.html). Matching is case-insensitive.

```text
-**/.git/**
-**/node_modules/**
-**/*.tmp
+public/**
```

- `-`: excludes matching paths.
- `+`: includes matching paths. If any `+` rule exists, paths that match no `+` rule are excluded.
- Exclusion takes precedence: a path matching both `+` and `-` is excluded.

Re-index after changing the rules. Existing index entries are not removed merely because the configuration changed.

## Search permissions

Search results are still constrained by the user's root path, path permissions, and hidden rules. A successful administrator test does not mean anonymous or normal users can see the same results.

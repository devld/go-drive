---
title: Path Patterns
description: Write go-drive path patterns with wildcards, recursive matches, exclusions, and ordering rules for permissions, indexing, and handlers.
lang: en
translation_key: path-patterns
---

# Path Patterns

Search filters, hidden attributes, job source paths, and event triggers in go-drive use doublestar-style path patterns. Paths have no leading `/` and use `/` as the separator.

| Pattern | Meaning |
| --- | --- |
| `*` | Matches any number of non-`/` characters within one directory level |
| `?` | Matches one non-`/` character |
| `**` | Spans zero or more directory levels |
| `[abc]` | Matches one character from the set |
| `{a,b}` | Matches alternatives where this syntax is supported |

Examples:

```text
a/*.js       matches a/x.js, but not a/sub/x.js
a/**/*.js    matches a/x.js and a/sub/x.js
**/.git/**   matches .git contents at any level
photos/202?  matches photos/2020 through photos/2029
```

Individual features add their own syntax around the pattern: each search-filter line also begins with `+` or `-`, while thumbnail mappings use `tag:<pattern>`.

Test wildcard delete and move jobs in a test directory first. `**` may match a very large directory tree.

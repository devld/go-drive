---
title: Path Attributes and Mounts
description: Use path attributes and mounts to customize file behavior, expose entries at virtual paths, and control inherited settings in go-drive.
lang: en
translation_key: path-attrs-mounts
---

# Path Attributes and Mounts

## Path attributes

Under **Admin → Path Attributes**, you can configure any virtual path with:

| Attribute | Effect |
| --- | --- |
| Password | Requires a path password before accessing a directory and its contents |
| Default sorting | Uses the selected sorting mode on entering the directory |
| Default mode | Selects list or thumbnail mode |
| Hidden patterns | Hides matching entries with path patterns |

Each attribute has its own **Recursive** switch. When enabled, it applies to child paths; a configuration on a more specific path can override its parent. The browser temporarily stores path passwords entered during the current session in `sessionStorage`; they must be entered again after the session is closed.

Hidden patterns are only an interface/path filtering mechanism and must not replace access control. Sensitive files should still have read access explicitly denied.

See [Path Pattern Reference](../reference/path-patterns.html) for pattern syntax.

## Create a mount

An administrator can choose **Mount to** from a file or directory's context menu and then select the destination. Mounted entries display an `@` marker visible only to administrators.

Mounting does not copy data. The same real entry can appear at multiple locations in the virtual directory tree. Recursive mounts are allowed, but avoid creating directory structures that are difficult to understand.

## Permissions and attributes

- A mount uses permissions at the mount location, not at the source path.
- Path attributes are also resolved from the virtual path the user is visiting.
- Removing a mount only removes the mount relationship, not the source file. Deleting a real entry under the mount modifies the source data.
- After a Drive is deleted or a path disappears, invalid mounts and permissions can be cleaned from the maintenance page.

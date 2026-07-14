---
title: Users, Groups, Root Paths, and Permissions
lang: en
translation_key: access-control
---

# Users, Groups, Root Paths, and Permissions

go-drive controls access in two layers: the visible root path first limits the directory range a user can see, then path permissions determine which paths within it are readable or writable.

## Users and groups

Create local users, change passwords, set personal root paths, and assign groups under **Admin → Users**. Create groups, set group root paths, and manage members under **Admin → Groups**.

The built-in `admin` group has administrative privileges and is not restricted by root paths. Do not add everyday users to `admin`.

LDAP users are created automatically on first login. When LDAP group synchronization is configured, their membership is updated at every login and cannot be manually overridden in the admin interface. See [LDAP configuration](../configuration/#ldap).

## Root-path precedence

Enter root paths relative to the virtual directory, without a leading `/`:

1. Members of the `admin` group are unrestricted.
2. A user's personal root path takes precedence when set.
3. Otherwise, go-drive selects the shallowest non-empty root path among the user's groups.
4. If all are empty, the global root directory is visible.
5. Signed-out visitors use **Site Settings → Anonymous user root path**.

For example, if a user belongs to groups rooted at `team` and `team/design`, the shallower `team` path is selected. A root path only changes the user's visible virtual entry point; it does not grant read or write permission automatically.

## Path permissions

Set permissions for a path from **Permissions** in the file context menu (long-press on mobile). Root-path permissions are under **Admin → Other**. A subject can be:

- `ANY`: everyone, including signed-out visitors.
- A user.
- A group.

Each rule contains read and write permissions and an allow or deny policy. Resolution follows these principles:

- More specific paths take precedence; when no rule exists, resolution walks up to the parent directory.
- At the same matching level, deny takes precedence over allow.
- User rules are more specific than group rules, and group rules are more specific than `ANY`.

After changing permissions, test in a private browser window and with a normal test account. Do not test only as an administrator, because administrators see different results.

## Anonymous access

To publish a read-only directory:

1. Restrict the anonymous root path to the public directory.
2. Grant `ANY` read-only access on that path.
3. Explicitly deny access to other sensitive paths.
4. Test file listings, contents, thumbnails, search, and WebDAV.

`web-dav.allow-anonymous` only controls whether WebDAV accepts anonymous requests; path permissions still apply.

## Interaction with mounts

Permissions are bound to virtual paths. After an entry is mounted elsewhere, permissions are matched again against its new path at the mount location; permissions from the original location do not follow automatically. See [Path Attributes and Mounts](./path-attrs-mounts.html).

---
title: Local File Drive
description: Map a server or NAS directory into go-drive while controlling its root path, filesystem access restrictions, and portability.
lang: en
translation_key: drive-local
---

# Local File Drive

A local Drive maps a filesystem directory on the server or in the container into go-drive.

## Restricted mode (default)

```yaml
free-fs: false
data-dir: ./data
```

Enter a relative path such as `photos` for the Drive's **Root directory**. The actual directory is `<data-dir>/local/photos`. The current version creates this directory automatically when the Drive is loaded for the first time; you do not need to create it manually.

In a container, `<data-dir>/local` is under the mounted `/app/data/local` by default, so its data is persisted with `go-drive-data`.

## Free filesystem mode

```yaml
free-fs: true
```

The root directory must then be an existing absolute path. go-drive does not create it automatically. Any member of the `admin` group can add a Drive pointing to any host path that the process can read or write, so enabling this option effectively grants administrators filesystem access within the permissions of the go-drive process.

A container can see only paths mounted into it. For example:

```yaml
services:
  go-drive:
    image: devld/go-drive
    volumes:
      - ./go-drive-data:/app/data
      - /srv/media:/media
```

Then set the Drive root directory to `/media`.

## Permissions and file modes

go-drive path permissions cannot override operating-system permissions. Make sure the process user has the required read and write access to the directory. When overwriting an existing file, go-drive tries to preserve its mode. New directories and files use the process defaults and umask.

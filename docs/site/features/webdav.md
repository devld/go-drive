---
title: Access Through WebDAV
description: Enable the go-drive WebDAV service and connect desktop, mobile, Linux, and command-line clients through a reverse proxy.
lang: en
translation_key: webdav-access
---

# Access Through WebDAV

Enable go-drive's built-in WebDAV service:

```yaml
web-dav:
  enabled: true
  prefix: /dav
  allow-anonymous: false
  max-cache-items: 1000
```

After restarting, the WebDAV address is:

```text
https://drive.example.com/dav/
```

Use a go-drive username and password with Basic Auth. User root paths, group root paths, and path read/write permissions all apply to WebDAV.

## Subpath deployment

If the site is hosted at `/drive/`:

```yaml
api-path: /drive
web-dav:
  enabled: true
  prefix: /drive/dav
  allow-anonymous: false
  max-cache-items: 1000
```

The client address is `https://example.com/drive/dav/`. The proxy must forward methods including `PROPFIND`, `PROPPATCH`, `MKCOL`, `COPY`, `MOVE`, `LOCK`, `UNLOCK`, `PUT`, and `DELETE`.

## Anonymous access

`allow-anonymous: true` accepts requests without Basic Auth, but does not bypass permissions. Before enabling it:

1. Set the anonymous user root path.
2. Grant the `ANY` subject only the required permissions on public paths.
3. Test directory listing, download, upload, overwrite, and delete with a client that has no credentials.

For a public read-only service, make sure `ANY` has no write permission.

## Cache and temporary space

WebDAV adapts virtual Drives to a filesystem interface. Some operations cache files in `temp-dir`; `max-cache-items` controls the number of file objects cached simultaneously. Monitor temporary-directory space during high concurrency or large-file operations.

## Client examples

```bash
# Linux davfs2 example
sudo mount -t davfs https://drive.example.com/dav/ /mnt/go-drive

# rclone
rclone config
# Select WebDAV and enter https://drive.example.com/dav/ as the URL
```

Windows mapped network drives impose additional system restrictions on Basic Auth, HTTPS certificates, and large files. If problems occur, verify the server with curl or rclone first, then investigate Windows WebClient settings.

> To connect another WebDAV service as storage for go-drive, see [WebDAV Storage Drive](../drives/webdav.html).

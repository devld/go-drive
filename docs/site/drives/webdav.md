---
title: WebDAV Storage Drive
description: Mount a remote WebDAV server as a go-drive storage backend and configure its URL, credentials, root path, and directory cache.
lang: en
translation_key: drive-webdav
---

# WebDAV Storage Drive

This page describes using another WebDAV service as a storage backend. To let clients access go-drive through WebDAV, see [WebDAV Service](../features/webdav.html).

| Field | Description |
| --- | --- |
| URL | WebDAV root URL, optionally including a remote path prefix |
| Username | Basic Auth username; may be empty |
| Password | Basic Auth password; may be empty |
| Cache TTL | Directory-entry cache time; zero or below disables caching |

Example: `https://dav.example.com/remote.php/dav/files/alice/`. The path in the URL becomes the remote root of this Drive.

File copies and moves within the same Drive use WebDAV `COPY` / `MOVE`; the operation fails if the remote service does not support it. Directory copies are usually performed recursively by go-drive. HTTPS is recommended to avoid transmitting Basic Auth credentials in plaintext.

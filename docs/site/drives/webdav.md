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
| Request headers | Additional headers sent to the remote WebDAV service |
| Cache TTL | Directory-entry cache time; zero or below disables caching |

Example: `https://dav.example.com/remote.php/dav/files/alice/`. The path in the URL becomes the remote root of this Drive.

Custom request headers are sent with every remote WebDAV request and override operation-specific values such as `Depth`, `Destination`, `Range`, and `Content-Type`. When a username is configured, go-drive applies Basic Auth afterward, replacing a custom `Authorization` header; without a username, custom `Authorization` is preserved. Connection-specific headers such as `Host` and `Content-Length` cannot be configured.

Header values are stored and returned as ordinary Drive configuration; they are not secret-masked. Use HTTPS when a header contains credentials or other sensitive data.

File copies and moves within the same Drive use WebDAV `COPY` / `MOVE`; the operation fails if the remote service does not support it. Directory copies are usually performed recursively by go-drive. HTTPS is recommended to avoid transmitting Basic Auth credentials in plaintext.

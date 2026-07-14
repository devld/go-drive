---
title: Drive Overview
lang: en
translation_key: drives
---

# Drive Overview

Add, edit, and enable Drives under **Admin → Drives**. After saving the configuration, click **Reload Drives** before the running virtual directory tree will use the new configuration.

## Built-in types

| Drive | Read/write | Native file move | Native file copy | Transfer behavior |
| --- | --- | --- | --- | --- |
| Local File | Yes | Yes | No | Copies are handled by a generic go-drive job |
| FTP | Yes | Yes | No | File contents pass through go-drive |
| SFTP | Yes | Yes | No | File contents pass through go-drive |
| WebDAV | Yes | Yes | Yes (files) | Uses remote `COPY`/`MOVE` |
| S3 | Yes | Yes | Yes (files) | Browser direct upload/download or forced proxy |
| OneDrive | Yes | Yes | Yes (files) | Browser direct upload/download or forced proxy |
| Google Drive | Yes | Yes | Yes (files) | Supports exporting native Google files |

“Native copy” means the remote service can copy a file within the same Drive. Directory copies, cross-Drive copies, and types without native copy support are recursively read and written by go-drive, consuming server bandwidth and temporary space.

## Cache

FTP, SFTP, WebDAV, S3, OneDrive, and Google Drive all provide `cache_ttl`. A value greater than zero caches directory entries to reduce remote requests. After a configuration change or a direct change in the external system, the interface may briefly show stale content.

- Normal interface operations try to invalidate the relevant cache entries.
- External changes do not notify go-drive.
- Clear a specific Drive under **Admin → Other → Clear Cache**.
- Set the value to zero or below to disable entry caching for that Drive, at the cost of more remote requests.

## Proxied uploads and downloads

S3, OneDrive, and similar types provide **Proxy upload** and **Proxy download** options:

- With proxying disabled, the browser can communicate directly with the cloud service, reducing go-drive traffic. Cloud CORS, domains, and hotlink protection must be configured correctly.
- With proxying enabled, traffic passes through go-drive. Deployment is simpler, but it consumes server bandwidth, connections, and temporary space.

Cross-Drive operations and background jobs may always pass through the server, so browser upload behavior alone is not enough to estimate bandwidth usage.

## Paths and mounts

The Drive name forms part of the virtual root directory. Any file or directory can also be mounted elsewhere; permissions for a mounted item are evaluated at the mount location. See [Path Attributes and Mounts](../administration/path-attrs-mounts.html).

## Script Drives

Extensions such as Dropbox and Qiniu are implemented in JavaScript and must be installed under **Admin → Other Drives**. Scripts and browser uploaders are trusted code. Review their source before installation; see [Script Drives](../extensions/script-drives.html).

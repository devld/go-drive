---
title: "go-drive: Self-Hosted File Manager for S3, WebDAV, and Cloud Storage"
titleTemplate: false
description: Manage local files, S3, WebDAV, FTP, SFTP, OneDrive, and Google Drive in one self-hosted go-drive server.
lang: en
translation_key: home
---

# go-drive

go-drive is a self-hosted file management server written in Go and Vue/TypeScript. It unifies local files, FTP, SFTP, WebDAV, S3, OneDrive, Google Drive, and script-based drives in one virtual directory tree, with permissions, search, WebDAV access, file buckets, thumbnails, and automated jobs.

> The default account on first startup is `admin` with password `123456`. Change it immediately after signing in, and read the [security guide](./configuration/security.html) before exposing the service publicly.

## Quick start

```bash
mkdir go-drive-data
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  --restart unless-stopped \
  devld/go-drive
```

Open `http://localhost:8089`. See [Installation and startup](./getting-started/) and [Upgrade, backup, and restore](./getting-started/upgrade-backup.html) for other installation methods and upgrade procedures.

## Feature overview

- Browse, upload, download, copy, move, rename, and delete files, with drag-and-drop, paste upload, chunked upload, and ZIP downloads.
- Users, groups, root-path isolation, and per-path read/write permissions.
- Image, audio, video, text, code, PDF, and configurable external viewers.
- Built-in image and text thumbnails; the official Docker image also includes libvips and ffmpeg.
- Filename search, WebDAV service, path mounts, and path attributes.
- File buckets with access controls, upload tokens, type/size limits, and hotlink protection.
- Copy, move, delete, and JavaScript jobs triggered by cron or file events.
- JavaScript extensions for new drive types and browser-side direct-upload adapters.

## Supported drives

| Type | Primary use | Important options |
| --- | --- | --- |
| Local files | Server or NAS filesystem | Restricted directory or `free-fs` |
| FTP | Traditional FTP servers | Concurrency, timeout, cache |
| SFTP | File access over SSH | Password/key, host key, root path |
| WebDAV | Any WebDAV-compatible server | URL, credentials, cache |
| S3 | Amazon S3 and compatible storage | Endpoint, region, path style, proxied transfer |
| OneDrive | Personal, organization, and SharePoint drives | Region, tenant, proxied transfer, cache |
| Google Drive | Personal and shared drives | Cache, proxied thumbnails |
| Dropbox, Qiniu, and others | JavaScript extensions | Script drive repository |

See the [drive overview](./drives/) for configuration details and capability limitations.

## Useful shortcuts

> On macOS, `Ctrl` refers to the <kbd>⌃ Control</kbd> key (not <kbd>⌘ Command</kbd>), and `Alt` refers to the <kbd>⌥ Option</kbd> key.

- `Ctrl` / `Shift` + click: select multiple entries.
- Copy files in your operating system and press `Ctrl+V`: paste-upload them.
- `Alt` + click a file: download it directly.
- Hold `Ctrl` while dragging an entry to copy it; hold `Shift` to create a path mount (administrators only).
- Use the file context menu (long-press on mobile) for permissions, mounts, rename, and other operations.

## Getting help

For OAuth, S3 CORS, reverse proxy, permission, cache, or indexing problems, start with [Troubleshooting](./troubleshooting/). If the problem remains, open a GitHub issue with the version, deployment method, relevant redacted configuration, and error logs.

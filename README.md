<div align="center">

# <img src="web/public/favicon.png" alt="go-drive logo" height="32" valign="middle"> go-drive

**A cross-platform, self-hosted file management server with a modern web UI.**

Manage local files and a wide range of cloud storage providers from a single, unified interface.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](web)
[![Docker](https://img.shields.io/badge/Docker-devld%2Fgo--drive-2496ED?logo=docker&logoColor=white)](https://hub.docker.com/r/devld/go-drive)
[![Docker Pulls](https://img.shields.io/docker/pulls/devld/go-drive?logo=docker&logoColor=white)](https://hub.docker.com/r/devld/go-drive)
[![GitHub Release](https://img.shields.io/github/v/release/devld/go-drive)](https://github.com/devld/go-drive/releases)

[Live Demo](https://demo.go-drive.top) · [Documentation](https://go-drive.top) · [Releases](https://github.com/devld/go-drive/releases)

English | [简体中文](README.zh-CN.md)

</div>

---

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Supported Drives](#supported-drives)
- [Quick Start](#quick-start)
  - [Docker](#docker)
  - [Docker Compose](#docker-compose)
  - [Prebuilt Binary](#prebuilt-binary)
- [Configuration](#configuration)
- [Building from Source](#building-from-source)
- [Contributing](#contributing)
- [License](#license)

## Overview

**go-drive** is a lightweight file-management server written in Go with a Vue/TypeScript frontend. It lets you browse, upload, organize, and share files across many different storage backends — local disks, FTP/SFTP, WebDAV, S3-compatible object storage, OneDrive, Google Drive, and more — all through one clean web interface.

The whole application ships as a single self-contained binary (the web UI and i18n assets are embedded), making it trivial to deploy on a server, NAS, or inside a container.

> Default credentials: user `admin`, password `123456`. **Change the password immediately after the first login.**

## Features

- **File management** — browse, copy, move, rename, and delete with drag-and-drop and paste-to-upload support.
- **Uploads & downloads** — chunked uploads for large files and zip packaging for batch downloads.
- **Permission control** — fine-grained, user- and group-based access control per path.
- **Path mounting** — mount any drive or subpath into a unified virtual tree.
- **Media preview** — image gallery powered by [PhotoSwipe](https://github.com/dimsemenov/PhotoSwipe), built-in music player, and video playback.
- **Editors** — text editing with [CodeMirror](https://github.com/codemirror/) and full code editing with [Monaco Editor](https://github.com/microsoft/monaco-editor).
- **Thumbnails** — generate thumbnails for images, text, video, and audio (pluggable handlers, optional `libvips`/`ffmpeg`).
- **Full-text search** — optional file search across mounted drives.
- **WebDAV access** — expose your drives over the WebDAV protocol.
- **Scheduled jobs** — cron-style background tasks powered by [gocron](https://github.com/go-co-op/gocron).
- **Extensible drives** — add new storage backends with JavaScript, no recompilation required.
- **Admin console** — manage drives, users, groups, permissions, and jobs from the browser.

## Supported Drives

| Drive | Notes |
| --- | --- |
| Local | Files on the host filesystem |
| FTP | FTP server |
| SFTP | SSH file transfer |
| WebDAV | Any WebDAV-compatible server |
| S3 | Amazon S3 and S3-compatible object storage |
| OneDrive | Microsoft OneDrive |
| Google Drive | Google Drive |
| Dropbox | Implemented as a scripted (JavaScript) drive |
| Qiniu | Qiniu Cloud, implemented as a scripted (JavaScript) drive |

See the [documentation](https://go-drive.top) for per-drive setup details.

## Quick Start

### Docker

```shell
mkdir go-drive-data
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  devld/go-drive
```

`go-drive-data` is the data directory. Mapping it outside the container ensures your data survives application upgrades. Once running, open <http://localhost:8089>.

The official Docker image bundles `ffmpeg` and `libvips`, and automatically enables the video/audio and high-performance image thumbnail handlers.

To customize the configuration, first extract the default `config.yml` from the image, then mount it back into the container:

```shell
# Extract config.yml from a throwaway container
cid=$(docker run -d devld/go-drive) && docker cp "$cid:/app/config.yml" . && docker stop "$cid" && docker rm "$cid"

# Run with both the data dir and the config file mapped outside the container
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  -v "$(pwd)/config.yml:/app/config.yml" \
  devld/go-drive
```

### Docker Compose

```yaml
services:
  go-drive:
    image: devld/go-drive
    container_name: go-drive
    ports:
      - "8089:8089"
    volumes:
      - ./go-drive-data:/app/data
    restart: unless-stopped
```

```shell
docker compose up -d
```

### Prebuilt Binary

Download the archive for your platform from the [Releases](https://github.com/devld/go-drive/releases) page, extract it, and run:

- **Linux**

```shell
./go-drive
```

- **Windows**

Run `go-drive.exe`.

By default go-drive reads `config.yml` from the working directory (if present) and listens on `:8089`. Useful flags:

```shell
./go-drive -c /path/to/config.yml   # use a specific config file
./go-drive -show-config             # print the parsed configuration
./go-drive -v                       # print version information
```

## Configuration

go-drive is configured through a YAML file. A fully documented reference lives in [`docs/config.yml`](docs/config.yml); copy it and adjust as needed. Selected options:

| Option | Default | Description |
| --- | --- | --- |
| `listen` | `:8089` | Address the HTTP server binds to |
| `data-dir` | `./data` | Directory for all application data (database, sessions, thumbnails, local files, etc.) |
| `temp-dir` | `data-dir/temp` | Temporary file directory |
| `max-concurrent-task` | `100` | Max concurrent tasks (copy, move, delete) |
| `free-fs` | `false` | Allow Local drives to use absolute paths (**security risk**) |
| `trusted-proxies` | _empty_ | Trusted proxy IPs/CIDRs for resolving the real client IP |
| `api-path` / `web-path` | _empty_ | Override API/static paths when served under a sub-path |

go-drive supports **SQLite** (default) and **MySQL**. WebDAV access, full-text search, thumbnail handlers, and reverse-proxy/sub-path deployment are all configured here too — see [`docs/config.yml`](docs/config.yml) and the [documentation](https://go-drive.top) for the full reference.

> ⚠️ Setting `free-fs: true` lets admin users browse the entire host filesystem through Local drives. Leave it `false` unless you understand the implications.

## Building from Source

**Prerequisites:** the Go version pinned in [`go.mod`](go.mod), Node.js 24, and a C toolchain (SQLite requires CGO).

```shell
git clone https://github.com/devld/go-drive.git
cd go-drive

# Build everything: frontend, backend, and release archive
make all
```

`make all` builds the Vue frontend (`web/dist`), embeds it together with the i18n assets into the Go binary, and produces a release archive under `build/`. For frontend-only development:

```shell
cd web
npm install
npm run dev        # start the dev server
npm run lint       # type-check + lint
npm run build-web  # production build
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository and create a feature branch.
2. For backend changes, run the relevant package tests (`go test`, and `go test -race` for concurrent code).
3. For frontend changes, run `npm run lint` and `npm run build-web` from `web/`.
4. Use [Conventional Commits](https://www.conventionalcommits.org/) with a short lowercase scope (e.g. `fix(server): ...`).
5. Open a pull request describing the motivation and behavior changes.

## License

Released under the [MIT License](LICENSE). Copyright © 2020 devld.

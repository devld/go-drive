---
title: Installation and Startup
description: Install go-drive with Docker, release packages, or source code, configure persistent data, and complete the first secure startup.
lang: en
translation_key: getting-started
---

# Installation and Startup

## Docker (recommended)

```bash
mkdir go-drive-data
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  --restart unless-stopped \
  devld/go-drive
```

`/app/data` contains the database, local drives, sessions, thumbnail cache, temporary files, and installed script drives. Map it outside the container or deleting the container will also delete the data.

The official image includes libvips and ffmpeg and automatically enables handlers for high-performance image thumbnails, video frames, and embedded audio artwork.

### Custom configuration

Extract the configuration bundled with the current image from a temporary container:

```bash
cid=$(docker create devld/go-drive)
docker cp "$cid:/app/config.yml" ./config.yml
docker rm "$cid"
```

Then mount the configuration file:

```bash
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  -v "$(pwd)/config.yml:/app/config.yml:ro" \
  --restart unless-stopped \
  devld/go-drive
```

## Docker Compose

```yaml
services:
  go-drive:
    image: devld/go-drive
    container_name: go-drive
    ports:
      - "8089:8089"
    volumes:
      - ./go-drive-data:/app/data
      # Uncomment to use a custom configuration
      # - ./config.yml:/app/config.yml:ro
    restart: unless-stopped
```

```bash
docker compose up -d
```

## Prebuilt packages

Download and extract the package for your platform from [GitHub Releases](https://github.com/devld/go-drive/releases):

```bash
# Linux
./go-drive

# Use a specific configuration file
./go-drive -c /path/to/config.yml
```

On Windows, run `go-drive.exe`. By default the application reads `config.yml` from its working directory. If the file does not exist, built-in defaults are used and the server listens on `:8089`.

## Build from source

Requirements:

- The Go version declared in `go.mod` (currently Go 1.26.4).
- Node.js 24 and npm.
- GNU Make.
- A C compiler toolchain; SQLite requires CGO.

```bash
git clone https://github.com/devld/go-drive.git
cd go-drive
BUILD_VERSION=dev make all
```

`make all` builds the frontend, Monaco Editor, backend, and release archive. The Web UI and i18n resources are embedded in the release binary, and output is written under `build/`.

For frontend-only development:

```bash
cd web
npm install
npm run dev
npm run lint
npm run build-web
```

## First sign-in

Open `http://localhost:8089` and use:

- Username: `admin`
- Password: `123456`

Change the password immediately. Before public deployment, also configure HTTPS, trusted proxies, least privilege, and backups; see the [security guide](../configuration/security.html).

## Next steps

- [Configuration](../configuration/)
- [Add drives](../drives/)
- [Users, groups, and permissions](../administration/access-control.html)
- [Upgrade, backup, and restore](./upgrade-backup.html)

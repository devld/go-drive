# AGENTS.md

This file applies to the entire repository. More specific `AGENTS.md` files
override it for their subtrees.

## Project overview

go-drive is a cross-platform file-management server with a Go backend and a
Vue/TypeScript frontend.

- `common/`: shared packages.
- `drive/`: drive implementations and common behavior.
- `script/`: JavaScript VM integration for script-backed drives.
- `server/`: HTTP APIs, jobs, search, thumbnails, and WebDAV.
- `storage/`: GORM models and data access.
- `web/`: Vue/TypeScript frontend.
- `docs/`: release configuration, translations, and operator examples.

## Build constraints

- Use the Go version in `go.mod`; frontend releases use Node.js 24.
- SQLite requires CGO. Production releases target Windows, glibc Linux, and
  musl Linux across multiple architectures. Keep backend changes portable and
  the Docker build Alpine-compatible.
- Avoid new platform-specific native dependencies unless the release matrix is
  updated with them.
- `make all` builds the backend, frontend, docs assets, and release archive; it
  may install frontend dependencies and is not a lightweight test command.
- Do not edit generated or bundled assets under `build/`, `web/dist/`, or
  dependency directories.

## Validation

For backend changes, run the relevant package tests and use `go test -race` for
concurrent code. Some integration tests require permission to bind localhost
ports.

For frontend changes, work from `web/` and run:

```sh
npm run lint
npm run build-web
```

Use `npm run build` only when validating Monaco Editor or complete release
assets.

## Project-specific behavior

- Update `docs/config.yml` when user-facing configuration changes.
- Keep configuration examples disabled unless required for normal operation.
- Thumbnail `shell` handlers use `/bin/sh` on Unix and `cmd.exe` on Windows;
  document platform assumptions for external commands.
- After transport or protocol failures, discard pooled resources unless their
  health is known.

## Commits

Use Conventional Commits with a short lowercase subsystem scope. Non-trivial
commits must include a body explaining the motivation, behavior changes, and
relevant compatibility implications.

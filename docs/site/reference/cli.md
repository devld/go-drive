---
title: Command-Line Reference
description: Use go-drive command-line flags to select configuration files, print version information, control startup, and run administrative operations.
lang: en
translation_key: cli
---

# Command-Line Reference

```text
-c <path>      Specify a YAML configuration file
-show-config   Print the parsed configuration with defaults filled in, then exit
-v             Print version, revision, and build time, then exit
```

Examples:

```bash
./go-drive -c /etc/go-drive/config.yml
./go-drive -c ./config.yml -show-config
./go-drive -v
```

Without `-c`, go-drive automatically reads `config.yml` from the working directory when it exists; otherwise it uses built-in defaults.

## Environment variables

```text
GO_DRIVE_DEBUG=1
```

Enables additional diagnostic behavior and logging. Use it only while troubleshooting and disable it afterward.

Common Make variables at build time:

```bash
BUILD_VERSION=dev BUILD_REV=$(git rev-parse HEAD) make all
```

`make all` requires the frontend toolchain and CGO. A plain `go build` does not produce a complete release package containing the Web UI.

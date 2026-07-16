---
title: Developing and Installing Script Drives
description: Install third-party script drives or develop custom go-drive storage adapters and browser-side direct-upload integrations in JavaScript.
lang: en
translation_key: script-drives
---

# Developing and Installing Script Drives

JavaScript Drives add storage backends without recompiling go-drive. Extensions for Dropbox, Qiniu, and other services use this mechanism.

They are intended for storage services with HTTP/HTTPS APIs that can map files and directories to paths. They are not a general protocol runtime: SMB/Samba, SFTP, FTP, local filesystems, and services that require raw sockets, native libraries, Node.js packages, or operating-system commands need a built-in Go Drive instead.

## Installation

Open **Admin → Other Drives**, refresh the repository, and select an extension to install. An extension usually contains:

- `<name>.js`: the server-side Drive implementation.
- `<name>-uploader.js`: an optional browser upload adapter.

Default repository:

```text
https://api.github.com/repos/devld/go-drive/contents/script-drives
```

It can be changed in the configuration:

```yaml
drives-dir: script-drives
drive-uploaders-dir: drive-uploaders
drive-repository-url: https://example.com/my-drives.json
```

A custom repository returns an array in the style of the GitHub Contents API, with at least `name` and `download_url`:

```json
[
  { "name": "example.js", "download_url": "https://example.com/example.js" },
  { "name": "example-uploader.js", "download_url": "https://example.com/example-uploader.js" }
]
```

After installation, create the corresponding type on the Drive management page and reload the Drives.

## Development entry points

Start with the current templates and definitions:

- [`docs/script-drive-template.js`](https://github.com/devld/go-drive/blob/master/docs/script-drive-template.js)
- [`docs/scripts/global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts)
- [`docs/scripts/env/drive.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/env/drive.d.ts)
- [`docs/drive-uploaders`](https://github.com/devld/go-drive/tree/master/docs/drive-uploaders)
- [`script-drives/AGENTS.md`](https://github.com/devld/go-drive/blob/master/script-drives/AGENTS.md): the complete implementation contract, API catalog, suitability guide, and end-to-end example for AI agents and developers.

The template uses TypeScript references for editor completion, but the runtime is still server-side JavaScript. An implementation should:

- Define a unique type name, display name, description, and configuration form.
- Implement the required `meta`, `get`, `list`, and `getReader` methods, then the write, upload, download, and thumbnail methods supported by the service.
- Return Unsupported from unavailable native `copy` operations so the dispatcher can stream-copy; note that `move` does not have a copy-and-delete fallback.
- Use the provided context and propagate cancellation through cancellable operations.
- Promptly close response bodies, readers, and remote connections.
- Never write tokens, passwords, or signed URLs to logs.

## Browser uploader

An uploader runs in the user's browser and can provide direct transfers to S3-like services. It must handle CORS, progress, cancellation, errors, and server-returned results. The server script and uploader are separate trust surfaces; review both when auditing an extension.

## Debugging

1. Install or edit the script in a test instance.
2. Use a dedicated test account and directory.
3. Enable `GO_DRIVE_DEBUG=1` for temporary diagnostic information.
4. Test empty files, large files, overwrite, directories, cancellation, network errors, and credential expiration separately.
5. Disable debug mode and reload the Drive when testing is complete.

Calls use a pool of concurrent script VMs. Ordinary mutable globals are not reliable shared state; only JSON-serializable `$` instance properties are synchronized between VMs. Scripts can access the network and Drive data, so do not treat the runtime as a sandbox for untrusted code.

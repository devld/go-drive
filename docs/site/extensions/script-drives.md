---
title: Developing and Installing Script Drives
lang: en
translation_key: script-drives
---

# Developing and Installing Script Drives

JavaScript Drives add storage backends without recompiling go-drive. Extensions for Dropbox, Qiniu, and other services use this mechanism.

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
- [`docs/scripts/drive.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/drive.d.ts)
- [`docs/drive-uploaders`](https://github.com/devld/go-drive/tree/master/docs/drive-uploaders)

The template uses TypeScript references for editor completion, but the runtime is still server-side JavaScript. An implementation should:

- Define a unique type name, display name, description, and configuration form.
- Correctly implement Get/List/Save/MakeDir/Copy/Move/Delete/Content.
- Return Unsupported for unavailable native operations so the scheduling layer can use its generic fallback.
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

The script VM isolates runtime state between calls, but scripts can still access the network and Drive data. Do not treat it as a sandbox for untrusted code.

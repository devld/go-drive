---
title: File Buckets
description: Publish controlled upload and download endpoints with go-drive file buckets, access tokens, type limits, and hotlink protection.
lang: en
translation_key: file-buckets
---

# File Buckets

A file bucket provides a programmatic upload and public-read endpoint. It is not the S3 protocol and does not use a normal user's login token. A common use case is to set up a self-hosted image hosting service for blogs and note-taking tools.

Create a bucket under **Admin → File Buckets**.

## Fields

| Field | Description |
| --- | --- |
| Name | Unique bucket name used in the URL |
| Target path | Virtual directory where files are actually written, without a leading `/` |
| File path template | Template used by the server to generate object paths |
| Upload key | Upload query parameter `t`; use an unguessable value |
| Download URL template | URL returned after a successful upload |
| Allow custom path | Lets callers specify a path in the URL |
| Maximum size | Size with a unit; zero or below means unlimited |
| Allowed types | List of MIME types, extensions, or wildcard MIME types |
| Allowed Referer | List of Referer hosts allowed to download |
| Cache duration | `Cache-Control` max-age for download responses |

## Upload

```bash
# multipart/form-data; the field must be named file
curl -f -F 'file=@photo.jpg' \
  'https://drive.example.com/f/assets?t=UPLOAD_TOKEN'

# Raw file stream
curl -f -X POST --data-binary '@photo.jpg' \
  'https://drive.example.com/f/assets?t=UPLOAD_TOKEN'
```

With custom paths enabled:

```bash
curl -f -F 'file=@photo.jpg' \
  'https://drive.example.com/f/assets/custom/2026/photo.jpg?t=UPLOAD_TOKEN'
```

The response body is the download URL and includes `X-File-Size` and `X-File-Mime`. The server detects the actual MIME type; do not rely only on the client-provided filename.

## Path template

Supported placeholders: `{year}`, `{month}`, `{date}`, `{hour}`, `{minute}`, `{second}`, `{millisecond}`, `{timestamp}`, `{rand}`, `{name}`, and `{ext}`.

Default:

```text
{year}{month}{date}/{name}-{rand}{ext}
```

## Download URL template

Supported placeholders are `{origin}`, `{bucket}`, and `{key}`. The default is:

```text
{origin}/f/{bucket}/{key}
```

The download endpoint is `GET`/`HEAD /f/<bucket-name>/<path>`. The default cache duration is `1d`; zero or below returns `no-cache`.

## Allowed types

Allowed types can be mixed:

```text
image/*,application/pdf,.jpg,.png
```

MIME types, file extensions, and wildcard MIME types (e.g. `image/*`) are all supported.

## Hotlink protection

The **Allowed Referer** field restricts which sites can download bucket files. Values are comma-separated hostnames. Wildcard subdomains are supported. An empty entry allows requests without a `Referer` header.

Example:

```text
example.com,*.example.com,cdn.other.com,
```

The trailing comma creates an empty entry, which allows requests that do not send a `Referer` header (e.g. direct browser visits). Without it, only requests from the listed domains are accepted.

Clients can forge `Referer` values, so this only discourages ordinary hotlinking and is not an authentication mechanism.

## Example: image hosting

A minimal image hosting setup for a blog or Markdown editor:

| Field | Value |
| --- | --- |
| Name | `images` |
| Target path | `blog/images` |
| File path template | `{year}{month}/{name}-{rand}{ext}` |
| Upload key | *(a random string)* |
| Allowed types | `image/*` |
| Maximum size | `10m` |
| Cache duration | `30d` |

Upload an image:

```bash
curl -f -F 'file=@screenshot.png' \
  'https://drive.example.com/f/images?t=YOUR_TOKEN'
```

The response body is the public URL, which can be used directly in Markdown:

```markdown
![screenshot](https://drive.example.com/f/images/202607/screenshot-a1b2c3d4e5f6.png)
```

Many Markdown editors (Typora, PicGo, etc.) can be configured to upload images via a custom command or API. Point them to the bucket upload URL with the upload token.

## Security recommendations

- Use a separate random upload token for each purpose and rotate it regularly.
- Restrict the target directory, file size, and MIME types.
- Keep custom paths disabled unless needed, so callers cannot overwrite predictable paths.
- File-bucket reads are public; do not rely on Referer protection for sensitive files.
- Use a long cache duration for immutable static assets. Use a short duration or versioned filenames for replaceable assets.

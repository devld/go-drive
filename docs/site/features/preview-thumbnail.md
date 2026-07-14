---
title: File Preview and Thumbnails
lang: en
translation_key: preview-thumbnail
---

# File Preview and Thumbnails

## File preview

Under **Admin → Site**, you can configure extension lists for:

- The CodeMirror text editor (files smaller than 128 KiB).
- The image gallery.
- Audio and video players.
- The Monaco code editor.
- External file previewers.

Each external previewer line has this format:

```text
<extension list> <URL template> <name>
```

PDF.js is included by default:

```text
pdf pdf.js/web/viewer.html?file={URL} PDF Viewer
```

Office example:

```text
docx,doc,xlsx,xls,pptx,ppt https://view.officeapps.live.com/op/embed.aspx?src={URL} Microsoft
```

The external service must be able to access `{URL}`. When enabled, a signed file URL is sent to the third party. Use a local previewer for internal or sensitive files.

**Maximum proxy-download size** and **Maximum ZIP size** limit the amount of file data the server will proxy or package. Use one-character units `b`, `k`, `m`, `g`, or `t`, for example `100m`. An empty value uses the application's default behavior.

## Thumbnail handlers

The configuration file supports:

- `image`: built-in image handler for jpg/jpeg/png/gif/webp.
- `text`: reads the beginning of a text file and generates an SVG.
- `shell`: runs an external program that writes the thumbnail to stdout.

The official Docker image includes:

- libvips: low-memory, high-performance image thumbnails, including WebP, TIFF, SVG, HEIC, and AVIF.
- ffmpeg: the first video frame and embedded audio artwork, output as WebP.

Extract `config.yml` from the Docker image to get the complete enabled handler templates.

## Shell handler

```yaml
thumbnail:
  handlers:
    - type: shell
      tags: media
      file-types: mp4,avi,mkv,mov,webm,mp3,flac,ogg,opus
      config:
        shell: ffmpeg -hide_banner -loglevel error -i - -an -frames:v 1 -vf scale=220:-1 -c:v libwebp -f webp -
        mime-type: image/webp
        write-content: true
        max-size: -1
        timeout: 10m
```

Unix uses `/bin/sh -c`; Windows uses `cmd.exe /D /S /C`. Scripts may span multiple lines and receive:

- `GO_DRIVE_ENTRY_TYPE`
- `GO_DRIVE_ENTRY_REAL_PATH`
- `GO_DRIVE_ENTRY_PATH`
- `GO_DRIVE_ENTRY_NAME`
- `GO_DRIVE_ENTRY_SIZE`
- `GO_DRIVE_ENTRY_MOD_TIME`
- `GO_DRIVE_ENTRY_URL`

Shell handlers run with the go-drive process's privileges; use only trusted commands. Setting `write-content: true` for remote files sends the entire content to stdin and may consume substantial network and CPU resources.

## Map paths to handlers

The site-setting format is:

```text
tag1,tag2:<path pattern>
```

go-drive first finds handlers by extension, then uses the path mapping's tag to select one. If no tag matches, it uses the default handler for that extension. Restart after changing handlers in the configuration file; changing only the interface mapping usually does not require a restart.

The thumbnail cache is stored under the data directory and controlled by `thumbnail.ttl`. Failures are cached briefly to avoid repeated resource use; restarting clears failure markers and allows another attempt.

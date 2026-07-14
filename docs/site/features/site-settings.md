---
title: Site Settings
description: Customize go-drive branding, anonymous access, file handling, appearance, injected styles, and other browser-facing site behavior.
lang: en
translation_key: site-settings
---

# Site Settings

**Admin → Site** stores database options, not `config.yml` settings. Changes take effect after saving and refreshing the page.

The web interface currently includes `en-US`, `zh-CN`, and `ko-KR`. It selects a language from the browser at startup and falls back to English if a matching language cannot be loaded. Released binaries embed these language resources and no longer use the old `lang-dir` configuration.

## Name, styles, and scripts

- **Application name** changes the page title and branding text.
- **Global CSS** provides theme overrides.
- **Injected script** executes JavaScript on every visitor's page.

A short CSS example:

```css
@media (prefers-color-scheme: light) {
  :root {
    --primary-bg-color: rgba(255, 255, 255, 0.9) !important;
    --secondary-bg-color: rgba(255, 255, 255, 0.75) !important;
  }

  body {
    background: #eef3f8;
  }
}
```

Internal class names and CSS variables are not a stable API. Check custom styles after upgrading. Injected scripts have the same browser privileges as the site and can read login state and page content; use only trusted code.

## File preview

You can separately configure extensions for text, images, audio, video, Monaco, and external previewers. See [File Preview and Thumbnails](./preview-thumbnail.html) for detailed formats and third-party privacy implications.

## Anonymous user root path

This restricts signed-out visitors to a virtual subdirectory. It does not grant read permission automatically; you must also configure path permissions for `ANY`. See [Users, Groups, Root Paths, and Permissions](../administration/access-control.html).

## Download options

- **Maximum proxy-download size** limits files that require the server to proxy them.
- **Maximum ZIP size** limits packaged downloads.

These limits protect server bandwidth, memory, and temporary space. With proxy downloading disabled for a cloud Drive, direct browser downloads may follow a different path. Cross-Drive and ZIP operations still consume server resources.

## Thumbnail mapping

Use `tag1,tag2:<path pattern>` to select configuration-file handlers for different directories. Handlers themselves are defined in `config.yml` and require a restart after changes. See [File Preview and Thumbnails](./preview-thumbnail.html) for mapping rules.

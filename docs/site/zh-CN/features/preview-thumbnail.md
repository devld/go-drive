---
title: 文件预览与缩略图
description: 为图片、视频、音频封面、文本、PDF、压缩包和 Office 文档配置 go-drive 文件预览器与缩略图处理器。
lang: zh-CN
translation_key: preview-thumbnail
source_hash: fab0e719a09b30c13874b2c1f5eac8dd588ca055aaafd13f88da12402067b57d
---

# 文件预览与缩略图

## 文件预览

“管理员 → 站点”可配置扩展名列表：

- CodeMirror 文本编辑器（文件小于 128 KiB）。
- 图片画廊。
- 音频播放器和视频播放器。
- Monaco 代码编辑器。
- 外部文件预览器。

外部预览器每行格式：

```text
<扩展名列表> <URL 模板> <名称>
```

默认内置 PDF.js：

```text
pdf pdf.js/web/viewer.html?file={URL} PDF Viewer
```

Office 示例：

```text
docx,doc,xlsx,xls,pptx,ppt https://view.officeapps.live.com/op/embed.aspx?src={URL} Microsoft
```

外部服务必须能访问 `{URL}`，启用后文件签名 URL 会发送给第三方。内网或敏感文件应使用本地预览器。

“代理下载最大大小”和“ZIP 最大大小”限制服务器代理读取和打包的文件规模。填写大小时使用 `b`、`k`、`m`、`g`、`t` 单字符单位，例如 `100m`；留空采用应用默认行为。

## 压缩包预览

ZIP、7z 和 RAR 可在浏览器中打开。你可以浏览其中的目录、下载单个文件，或将所选内容打包为 zip。

大小和缓存限制写在 `config.yml` 的 `archive` 下，包括 `max-size`、`max-member-size`、`max-entries`、控制打包 zip 可下载时长的 `pack-ttl`，以及 `concurrent`（压缩包预览和 zip 打包各自默认 4）。详见[配置文件参考](../configuration/)。

## 缩略图处理器

配置文件支持：

- `image`：内置图片处理器，支持 jpg/jpeg/png/gif/webp。
- `text`：读取文本开头生成 SVG。
- `shell`：运行外部程序，输出缩略图到 stdout。

部分 Drive 会在条目上设置 `hasThumbnail`（OneDrive、Google Drive，以及实现了 `getThumbnail` 的脚本 Drive）。即使扩展名不在 handler 列表中，界面也会请求缩略图；处理器会调用 `Entry.Thumbnail`，而不走本地 image、text 或 shell handler。

官方 Docker 镜像包含：

- libvips：低内存、高性能图片缩略图，含 WebP、TIFF、SVG、HEIC、AVIF 等格式。
- ffmpeg：视频首帧和音频内嵌封面，输出 WebP。

从 Docker 镜像提取 `config.yml` 可取得启用后的完整 handler 模板。

## Shell 处理器

```yaml
thumbnail:
  handlers:
    - type: shell
      file-types: mp4,avi,mkv,mov,webm,mp3,flac,ogg,opus
      config:
        shell: ffmpeg -hide_banner -loglevel error -i - -an -frames:v 1 -vf scale=220:-1 -c:v libwebp -f webp -
        mime-type: image/webp
        write-content: true
        max-size: -1
        timeout: 10m
```

Unix 使用 `/bin/sh -c`，Windows 使用 `cmd.exe /D /S /C`。脚本支持多行，并收到：

- `GO_DRIVE_ENTRY_TYPE`
- `GO_DRIVE_ENTRY_REAL_PATH`
- `GO_DRIVE_ENTRY_PATH`
- `GO_DRIVE_ENTRY_NAME`
- `GO_DRIVE_ENTRY_SIZE`
- `GO_DRIVE_ENTRY_MOD_TIME`
- `GO_DRIVE_ENTRY_URL`

Shell 处理器以 go-drive 进程权限执行，只能使用可信命令。对远端文件设置 `write-content: true` 会把完整内容传入 stdin，可能消耗大量网络和 CPU。

缩略图缓存位于数据目录下，受 `thumbnail.ttl` 控制。确定性的失败结果会被短期缓存以避免重复消耗，并会在重启后保留，直到 TTL 过期。

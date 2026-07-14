---
title: 文件预览与缩略图
description: 为图片、视频、音频封面、文本、PDF 和 Office 文档配置 go-drive 文件预览器与缩略图处理器。
lang: zh-CN
translation_key: preview-thumbnail
source_hash: a294d6c97d2b95e9050b218a1ae1b7a5f8b270a41cdf5d195128c5364d7fbada
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

## 缩略图处理器

配置文件支持：

- `image`：内置图片处理器，支持 jpg/jpeg/png/gif/webp。
- `text`：读取文本开头生成 SVG。
- `shell`：运行外部程序，输出缩略图到 stdout。

官方 Docker 镜像包含：

- libvips：低内存、高性能图片缩略图，含 WebP、TIFF、SVG、HEIC、AVIF 等格式。
- ffmpeg：视频首帧和音频内嵌封面，输出 WebP。

从 Docker 镜像提取 `config.yml` 可取得启用后的完整 handler 模板。

## Shell 处理器

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

Unix 使用 `/bin/sh -c`，Windows 使用 `cmd.exe /D /S /C`。脚本支持多行，并收到：

- `GO_DRIVE_ENTRY_TYPE`
- `GO_DRIVE_ENTRY_REAL_PATH`
- `GO_DRIVE_ENTRY_PATH`
- `GO_DRIVE_ENTRY_NAME`
- `GO_DRIVE_ENTRY_SIZE`
- `GO_DRIVE_ENTRY_MOD_TIME`
- `GO_DRIVE_ENTRY_URL`

Shell 处理器以 go-drive 进程权限执行，只能使用可信命令。对远端文件设置 `write-content: true` 会把完整内容传入 stdin，可能消耗大量网络和 CPU。

## 路径到处理器的映射

站点设置中的格式是：

```text
tag1,tag2:<路径模式>
```

先按扩展名寻找处理器，再用路径映射的 tag 选择；没有 tag 匹配时使用该扩展名的默认处理器。修改配置文件中的处理器后需要重启；只修改界面映射通常不需要重启。

缩略图缓存位于数据目录下，受 `thumbnail.ttl` 控制。失败结果会被短期缓存以避免重复消耗；重启会清理失败标记并允许重试。

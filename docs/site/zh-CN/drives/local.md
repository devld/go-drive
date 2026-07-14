---
title: 本地文件 Drive
description: 将服务器或 NAS 目录映射到 go-drive，并控制根路径、文件系统访问限制和跨平台兼容性。
lang: zh-CN
translation_key: drive-local
source_hash: ef923f3d728ca74145657e63b31faaf12ab37a0371b46a6990b09cd0d3857229
---

# 本地文件 Drive

本地 Drive 将服务器或容器中的文件系统目录映射到 go-drive。

## 受限模式（默认）

```yaml
free-fs: false
data-dir: ./data
```

Drive 的“根目录”填写相对路径，例如 `photos`，实际目录是 `<data-dir>/local/photos`。当前版本会在首次加载 Drive 时自动创建这个目录，不需要提前手工创建。

容器内的 `<data-dir>/local` 默认位于已挂载的 `/app/data/local`，因此数据会随 `go-drive-data` 持久化。

## 自由文件系统模式

```yaml
free-fs: true
```

此时根目录必须是已经存在的绝对路径。go-drive 不会自动创建它。任何 `admin` 组成员都可以添加指向主机任意可读写路径的 Drive，因此此选项等同于授予管理员 go-drive 进程权限范围内的文件系统访问能力。

容器内只能看到已经挂载进容器的路径。例如：

```yaml
services:
  go-drive:
    image: devld/go-drive
    volumes:
      - ./go-drive-data:/app/data
      - /srv/media:/media
```

然后将 Drive 根目录设置为 `/media`。

## 权限和文件模式

go-drive 的路径权限不能突破操作系统权限。确保运行用户对目录拥有所需读写权限。覆盖已有文件时会尽量保留原文件模式；新目录和文件使用进程默认权限及 umask。

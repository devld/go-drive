---
title: WebDAV 存储 Drive
lang: zh-CN
translation_key: drive-webdav
source_hash: 0641da688c8ce2fcf328c0bd5bff381d1b2df1e8ba01a1d3936d22ea028d0311
---

# WebDAV 存储 Drive

本页介绍把另一个 WebDAV 服务作为存储后端。要让客户端通过 WebDAV 访问 go-drive，请看[WebDAV 服务](../features/webdav.html)。

| 字段 | 说明 |
| --- | --- |
| URL | WebDAV 根 URL，可包含远端路径前缀 |
| 用户名 | Basic Auth 用户名，可留空 |
| 密码 | Basic Auth 密码，可留空 |
| 缓存 TTL | 目录项缓存时间；不大于零关闭缓存 |

示例：`https://dav.example.com/remote.php/dav/files/alice/`。URL 中的路径会作为该 Drive 的远端根路径。

同一 Drive 内的文件复制和移动使用 WebDAV `COPY` / `MOVE`；远端服务不支持时操作会失败。目录复制通常由 go-drive 递归执行。推荐 HTTPS，避免 Basic Auth 凭据明文传输。

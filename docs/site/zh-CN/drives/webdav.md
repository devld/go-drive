---
title: WebDAV 存储 Drive
description: 将远端 WebDAV 服务器挂载为 go-drive 存储后端，并配置 URL、账号、根路径和目录缓存。
lang: zh-CN
translation_key: drive-webdav
source_hash: 13d5ec641043caa11e0d7c0fa60ba98a194eb1cbe77869c5433028ebe47ca6f1
---

# WebDAV 存储 Drive

本页介绍把另一个 WebDAV 服务作为存储后端。要让客户端通过 WebDAV 访问 go-drive，请看[WebDAV 服务](../features/webdav.html)。

| 字段 | 说明 |
| --- | --- |
| URL | WebDAV 根 URL，可包含远端路径前缀 |
| 用户名 | Basic Auth 用户名，可留空 |
| 密码 | Basic Auth 密码，可留空 |
| 自定义请求头 | 发往远端 WebDAV 服务的额外请求头 |
| 缓存 TTL | 目录项缓存时间；不大于零关闭缓存 |

示例：`https://dav.example.com/remote.php/dav/files/alice/`。URL 中的路径会作为该 Drive 的远端根路径。

自定义请求头会随每个远端 WebDAV 请求发出，并覆盖 `Depth`、`Destination`、`Range`、`Content-Type` 等具体操作生成的值。配置用户名时，go-drive 会在其后应用 Basic Auth，覆盖自定义 `Authorization`；未配置用户名时则保留自定义 `Authorization`。不能配置 `Host`、`Content-Length` 等连接控制请求头。

请求头值会作为普通 Drive 配置存储和返回，不会进行秘密遮罩。请求头包含凭据或其他敏感数据时请使用 HTTPS。

同一 Drive 内的文件复制和移动使用 WebDAV `COPY` / `MOVE`；远端服务不支持时操作会失败。目录复制通常由 go-drive 递归执行。推荐 HTTPS，避免 Basic Auth 凭据明文传输。

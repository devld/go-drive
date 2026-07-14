---
title: FTP Drive
lang: zh-CN
translation_key: drive-ftp
source_hash: 6e66b4e0501147b2109e9d5632fc869ce2a2267eb7a1b7e3738bca85fe8d0318
---

# FTP Drive

| 字段 | 说明 | 默认值 |
| --- | --- | --- |
| 主机 | FTP 服务器主机名或 IP，不含端口 | 必填 |
| 端口 | FTP 端口 | `21` |
| 用户名 | 留空时使用匿名用户 | `anonymous` |
| 密码 | 留空时使用匿名密码 | `anonymous` |
| 并发数 | 连接池最大并发 | `5` |
| 超时 | 连接/操作超时，Go duration 格式 | `5s` |
| 缓存 TTL | 目录项缓存时间；不大于零关闭缓存 | 关闭 |

FTP 不提供传输加密，公网环境优先使用 SFTP。文件上传、下载和通用复制会经过 go-drive；同一 FTP Drive 内移动/重命名使用远端 rename。

如果服务器限制连接数，将并发数调低。出现列表内容陈旧时清除 Drive 缓存或缩短缓存 TTL。

---
title: Drive 总览
description: 比较 go-drive 支持的存储后端、能力差异、配置要求及浏览器直传和直下限制。
lang: zh-CN
translation_key: drives
source_hash: a8b26b8764f063973ca0c6b01b080f34e81b6b44983dd836c8cae5a0ddb89eab
---

# Drive 总览

在“管理员 → 盘”中添加、编辑和启用 Drive。保存配置后点击“重新加载盘”，运行中的虚拟目录树才会使用新配置。

## 内置类型

| Drive | 读写 | 原生文件移动 | 原生文件复制 | 传输特点 |
| --- | --- | --- | --- | --- |
| 本地文件 | 是 | 是 | 否 | 复制由 go-drive 通用任务完成 |
| FTP | 是 | 是 | 否 | 文件内容经过 go-drive |
| SFTP | 是 | 是 | 否 | 文件内容经过 go-drive |
| WebDAV | 是 | 是 | 是（文件） | 使用远端 `COPY`/`MOVE` |
| S3 | 是 | 是 | 是（文件） | 可由浏览器直传/直下或强制代理 |
| OneDrive | 是 | 是 | 是（文件） | 可由浏览器直传/直下或强制代理 |
| Google Drive | 是 | 是 | 是（文件） | 支持 Google 原生文档导出 |

“原生复制”表示同一 Drive 内可以让远端服务完成文件复制。目录复制、跨 Drive 复制或不支持原生复制的类型会由 go-drive 递归读取和写入，消耗服务器带宽和临时空间。

## 缓存

FTP、SFTP、WebDAV、S3、OneDrive 和 Google Drive 都提供 `cache_ttl`。大于零时会缓存目录项以减少远端请求；配置变更或外部系统直接修改文件后，界面可能短时间显示旧内容。

- 日常界面操作会尽量使相关缓存失效。
- 外部修改不会通知 go-drive。
- 可在“管理员 → 其他 → 清除缓存”清理指定 Drive。
- 设为不大于零可关闭该 Drive 的条目缓存，但远端请求会增加。

## 代理上传和下载

S3、OneDrive 等提供“代理上传/代理下载”选项：

- 关闭代理时，浏览器可直接与云服务通信，减少 go-drive 流量；必须正确配置云端 CORS、域名和防盗链。
- 开启代理时，流量经过 go-drive，部署更简单，但会消耗服务器带宽、连接和临时空间。

跨 Drive 操作和后台任务始终可能经过服务器，不能只根据网页上传方式估算带宽。

## 路径和挂载

Drive 名称构成虚拟根目录的一部分。还可以把任意文件或目录挂载到其他位置；挂载后的权限按挂载位置计算。详见[路径属性与挂载](../administration/path-attrs-mounts.html)。

## 脚本 Drive

Dropbox、七牛云等扩展使用 JavaScript 实现，需要从“管理员 → 其他盘”安装。脚本和浏览器上传器都属于受信任代码，安装前应审核来源，见[脚本 Drive](../extensions/script-drives.html)。

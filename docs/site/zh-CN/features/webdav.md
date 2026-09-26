---
title: 通过 WebDAV 访问
description: 启用 go-drive WebDAV 服务，并通过反向代理连接桌面、移动端、Linux 和命令行客户端。
lang: zh-CN
translation_key: webdav-access
source_hash: f5bee153ee993a8d946bd35999014e24afae1e49f2b1259543c0cedcce5c653c
---

# 通过 WebDAV 访问

启用 go-drive 自带的 WebDAV 服务：

```yaml
web-dav:
  enabled: true
  prefix: /dav
  allow-anonymous: false
```

重启后，WebDAV 地址为：

```text
https://drive.example.com/dav/
```

使用 go-drive 用户名和密码进行 Basic Auth。用户根路径、组根路径和路径读写权限均会应用到 WebDAV。

## 子路径部署

如果站点位于 `/drive/`：

```yaml
api-path: /drive
web-dav:
  enabled: true
  prefix: /drive/dav
  allow-anonymous: false
```

客户端地址是 `https://example.com/drive/dav/`。代理必须转发 `PROPFIND`、`PROPPATCH`、`MKCOL`、`COPY`、`MOVE`、`LOCK`、`UNLOCK`、`PUT` 和 `DELETE` 等方法。

## 匿名访问

`allow-anonymous: true` 允许没有 Basic Auth 的请求进入，但不会绕过权限。启用前：

1. 设置匿名用户根路径。
2. 只给公开路径的 `ANY` 主体授予必要权限。
3. 用无凭据客户端测试列目录、下载、上传、覆盖和删除。

公开只读服务应确保 `ANY` 没有写权限。

## 缓存和临时空间

WebDAV 将虚拟 Drive 适配成文件系统接口，并通过共用的 DriveFS 源文件缓存读取文件内容。`vfs.cache-items` 和 `vfs.cache-size` 限制该缓存。大量并发或大文件操作时监控临时目录空间。

## 客户端示例

```bash
# Linux davfs2 示例
sudo mount -t davfs https://drive.example.com/dav/ /mnt/go-drive

# rclone
rclone config
# 类型选择 WebDAV，URL 填 https://drive.example.com/dav/
```

Windows 映射网络驱动器对 Basic Auth、HTTPS 证书和大文件有额外系统限制；出现问题时先用 curl 或 rclone 验证服务器，再排查 Windows WebClient 配置。

> 将其他 WebDAV 服务接入 go-drive 的说明见 [WebDAV 存储 Drive](../drives/webdav.html)。

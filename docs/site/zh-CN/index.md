---
title: go-drive
lang: zh-CN
translation_key: home
source_hash: 190b480b12b5ee27f0b568d87f8958011a1212edd5eb5f0b7da1607a9da48a4e
---

# go-drive

go-drive 是一个使用 Go 和 Vue/TypeScript 编写的自托管文件管理服务器。它将本地磁盘、FTP、SFTP、WebDAV、S3、OneDrive、Google Drive 和脚本扩展盘统一到一棵虚拟目录树中，并提供权限、搜索、WebDAV、文件桶、缩略图和自动任务等能力。

> 首次启动的默认账号是 `admin`，密码是 `123456`。首次登录后请立即修改密码，并在公开部署前阅读[安全指南](./configuration/security.html)。

## 快速开始

```bash
mkdir go-drive-data
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  --restart unless-stopped \
  devld/go-drive
```

打开 `http://localhost:8089`。更多运行方式和升级方法见[安装与启动](./getting-started/)和[升级、备份与恢复](./getting-started/upgrade-backup.html)。

## 功能概览

- 浏览、上传、下载、复制、移动、重命名和删除文件，支持拖拽、粘贴、分片上传和 ZIP 打包下载。
- 用户、用户组、根路径隔离和按路径读写权限。
- 图片、音频、视频、文本、代码、PDF 与可配置的外部预览器。
- 内置图片/文本缩略图；官方 Docker 镜像还提供 libvips 和 ffmpeg。
- 全文文件名搜索、WebDAV 服务、路径挂载和路径属性。
- 带访问控制、上传 Token、类型/大小限制和防盗链的文件桶。
- Cron 或文件事件触发的复制、移动、删除和 JavaScript 任务。
- 使用 JavaScript 安装新的 Drive 和浏览器直传适配器。

## 支持的 Drive

| 类型 | 主要用途 | 重要选项 |
| --- | --- | --- |
| 本地文件 | 服务器或 NAS 文件系统 | 受限目录或 `free-fs` |
| FTP | 传统 FTP 服务 | 并发、超时、缓存 |
| SFTP | SSH 文件服务 | 密码/私钥、主机密钥、根路径 |
| WebDAV | 其他 WebDAV 服务 | URL、账号、缓存 |
| S3 | AWS S3 及兼容服务 | endpoint、region、path-style、代理传输 |
| OneDrive | 个人盘、组织盘、SharePoint | 区域、tenant、代理传输、缓存 |
| Google Drive | 个人盘和共享盘 | 缓存、缩略图代理 |
| Dropbox、七牛云等 | JavaScript 扩展 | 脚本 Drive 仓库 |

具体配置和能力限制见 [Drive 总览](./drives/)。

## 常用操作提示

> macOS 上 `Ctrl` 指 <kbd>⌃ Control</kbd> 键（不是 <kbd>⌘ Command</kbd>），`Alt` 指 <kbd>⌥ Option</kbd> 键。

- `Ctrl` / `Shift` + 单击：多选。
- 从系统复制文件后按 `Ctrl+V`：粘贴上传。
- `Alt` + 单击文件：直接下载。
- 拖拽条目时按 `Ctrl`：复制；按 `Shift`：创建路径挂载（管理员）。
- 文件右键菜单（移动端长按）：权限、挂载、重命名等操作。

## 获取帮助

遇到 OAuth、S3 CORS、反向代理、权限、缓存或索引问题时，先查看[故障排查](./troubleshooting/)。如果问题仍然存在，请在 GitHub issue 中附上版本、部署方式、相关配置（隐藏密钥）和错误日志。

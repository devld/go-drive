<div align="center">

# <img src="web/public/favicon.png" alt="go-drive logo" height="32" valign="middle"> go-drive

**一个跨平台、可自托管的文件管理服务器，配备现代化的 Web 界面。**

通过统一的界面管理本地文件以及多种云存储服务。

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](go.mod)
[![Vue](https://img.shields.io/badge/Vue-3-42b883?logo=vue.js&logoColor=white)](web)
[![Docker](https://img.shields.io/badge/Docker-devld%2Fgo--drive-2496ED?logo=docker&logoColor=white)](https://hub.docker.com/r/devld/go-drive)
[![Docker Pulls](https://img.shields.io/docker/pulls/devld/go-drive?logo=docker&logoColor=white)](https://hub.docker.com/r/devld/go-drive)
[![GitHub Release](https://img.shields.io/github/v/release/devld/go-drive)](https://github.com/devld/go-drive/releases)

[在线演示](https://demo.go-drive.top) · [文档](https://go-drive.top) · [版本发布](https://github.com/devld/go-drive/releases)

[English](README.md) | 简体中文

</div>

---

## 目录

- [项目简介](#项目简介)
- [功能特性](#功能特性)
- [支持的存储类型](#支持的存储类型)
- [快速开始](#快速开始)
  - [Docker](#docker)
  - [Docker Compose](#docker-compose)
  - [预编译二进制文件](#预编译二进制文件)
- [配置](#配置)
- [从源码构建](#从源码构建)
- [参与贡献](#参与贡献)
- [许可证](#许可证)

## 项目简介

**go-drive** 是一个使用 Go 编写、前端基于 Vue/TypeScript 的轻量级文件管理服务器。它让你能够通过一个简洁的 Web 界面，在多种存储后端之间浏览、上传、整理和分享文件 —— 包括本地磁盘、FTP/SFTP、WebDAV、S3 兼容对象存储、OneDrive、Google Drive 等等。

整个应用以单个自包含的二进制文件分发（Web 界面与 i18n 资源已内嵌），因此可以非常方便地部署在服务器、NAS 或容器中。

> 默认账号：用户名 `admin`，密码 `123456`。**请在首次登录后立即修改密码。**

## 功能特性

- **文件管理** —— 浏览、复制、移动、重命名、删除，支持拖拽操作以及拖拽/粘贴上传。
- **上传与下载** —— 大文件分片上传，批量文件打包为 zip 下载。
- **访问控制** —— 通过虚拟根路径隔离用户、用户组和匿名访客，并提供细粒度的路径读写权限。
- **LDAP 认证** —— 支持 LDAP/LDAPS、用户首次登录自动创建以及用户组自动同步。
- **路径挂载** —— 将任意存储或子路径挂载到统一的虚拟目录树中。
- **文件预览** —— 基于 [PhotoSwipe](https://github.com/dimsemenov/PhotoSwipe) 的图片画廊、内置音视频播放器和 PDF 查看器，并支持配置外部预览器。
- **编辑器** —— 使用 [CodeMirror](https://github.com/codemirror/) 进行文本编辑，使用 [Monaco Editor](https://github.com/microsoft/monaco-editor) 进行完整的代码编辑。
- **缩略图** —— 为图片、文本、视频和音频生成缩略图（处理器可插拔，可选 `libvips`/`ffmpeg`）。
- **文件名搜索** —— 可选的跨挂载存储文件名索引与搜索。
- **WebDAV 访问** —— 通过 WebDAV 协议访问你的存储。
- **自动化任务** —— 通过 cron 或文件事件触发复制、移动、删除、流程和 JavaScript 动作，并记录执行历史与日志。
- **文件桶** —— 提供带上传 Token 的程序化上传和公开读取端点，支持路径模板、类型/大小限制、缓存与防盗链。
- **可扩展存储** —— 使用 JavaScript 添加新的存储后端，无需重新编译。
- **管理控制台** —— 在浏览器中管理存储、用户、用户组、权限和任务。

## 支持的存储类型

| 存储 | 说明 |
| --- | --- |
| 本地文件 | 主机文件系统上的文件 |
| FTP | FTP 服务器 |
| SFTP | 基于 SSH 的文件传输 |
| WebDAV | 任意兼容 WebDAV 的服务器 |
| S3 | Amazon S3 及 S3 兼容对象存储 |
| OneDrive | 微软 OneDrive |
| Google Drive | 谷歌云端硬盘 |
| Dropbox | 通过脚本（JavaScript）实现 |
| 七牛云 | 通过脚本（JavaScript）实现 |

各存储类型的具体配置方式请参见[文档](https://go-drive.top)。

## 快速开始

### Docker

```shell
mkdir go-drive-data
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  devld/go-drive
```

`go-drive-data` 是数据目录。将其映射到容器之外，可确保在应用升级后数据得以保留。启动后，访问 <http://localhost:8089>。

官方 Docker 镜像已内置 `ffmpeg` 和 `libvips`，并自动启用视频/音频以及高性能图片缩略图处理器。

如需自定义配置，可先从镜像中提取默认的 `config.yml`，再将其映射回容器：

```shell
# 通过一个临时容器提取 config.yml
cid=$(docker run -d devld/go-drive) && docker cp "$cid:/app/config.yml" . && docker stop "$cid" && docker rm "$cid"

# 同时将数据目录和配置文件映射到容器外
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  -v "$(pwd)/config.yml:/app/config.yml" \
  devld/go-drive
```

### Docker Compose

```yaml
services:
  go-drive:
    image: devld/go-drive
    container_name: go-drive
    ports:
      - "8089:8089"
    volumes:
      - ./go-drive-data:/app/data
    restart: unless-stopped
```

```shell
docker compose up -d
```

### 预编译二进制文件

在 [Releases](https://github.com/devld/go-drive/releases) 页面下载对应平台的压缩包，解压后运行：

- **Linux**

```shell
./go-drive
```

- **Windows**

直接运行 `go-drive.exe` 即可。

默认情况下，go-drive 会读取工作目录中的 `config.yml`（如果存在），并监听 `:8089`。常用参数：

```shell
./go-drive -c /path/to/config.yml   # 使用指定的配置文件
./go-drive -show-config             # 打印解析后的配置
./go-drive -v                       # 打印版本信息
```

## 配置

go-drive 通过 YAML 文件进行配置。完整且带注释的参考配置位于 [`docs/config.yml`](docs/config.yml)，可复制后按需修改。部分常用选项：

| 选项 | 默认值 | 说明 |
| --- | --- | --- |
| `listen` | `:8089` | HTTP 服务监听地址 |
| `data-dir` | `./data` | 所有应用数据目录（数据库、会话、缩略图、本地文件等） |
| `temp-dir` | `data-dir/temp` | 临时文件目录 |
| `max-concurrent-task` | `100` | 最大并发任务数（复制、移动、删除） |
| `free-fs` | `false` | 允许本地存储使用绝对路径（**存在安全风险**） |
| `trusted-proxies` | _空_ | 用于解析真实客户端 IP 的可信代理 IP/CIDR |
| `api-path` / `web-path` | _空_ | 在子路径下提供服务时覆盖 API/静态资源路径 |

go-drive 支持 **SQLite**（默认）和 **MySQL**。WebDAV 访问、全文搜索、缩略图处理器以及反向代理/子路径部署等也都在此配置 —— 完整参考请见 [`docs/config.yml`](docs/config.yml) 和[文档](https://go-drive.top)。

> ⚠️ 设置 `free-fs: true` 会允许管理员用户通过本地存储浏览整个主机文件系统。除非你完全了解其影响，否则请保持为 `false`。

## 从源码构建

**前置要求：** [`go.mod`](go.mod) 中指定的 Go 版本、Node.js 24，以及 C 工具链（SQLite 需要 CGO）。

```shell
git clone https://github.com/devld/go-drive.git
cd go-drive

# 构建全部：前端、后端及发布压缩包
make all
```

`make all` 会构建 Vue 前端（`web/dist`），将其与 i18n 资源一起内嵌到 Go 二进制文件中，并在 `build/` 下生成发布压缩包。仅前端开发时：

```shell
cd web
npm install
npm run dev        # 启动开发服务器
npm run lint       # 类型检查 + lint
npm run build-web  # 生产构建
```

## 参与贡献

欢迎贡献代码！请遵循以下流程：

1. Fork 仓库并创建特性分支。
2. 后端改动请运行相关包的测试（`go test`，并发代码使用 `go test -race`）。
3. 前端改动请在 `web/` 目录下运行 `npm run lint` 和 `npm run build-web`。
4. 使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范，并带上简短的小写 scope（例如 `fix(server): ...`）。
5. 提交 Pull Request，说明改动的动机与行为变化。

## 许可证

基于 [MIT 许可证](LICENSE) 发布。版权所有 © 2020 devld。

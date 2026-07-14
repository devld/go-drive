---
title: 安装与启动
description: 使用 Docker、发行包或源码安装 go-drive，配置持久化数据，并完成首次安全启动。
lang: zh-CN
translation_key: getting-started
source_hash: 60322ee619d5ef80db193a6c1b9af511b364d84fa6873c7c1ca85f7a1bc3e407
---

# 安装与启动

## Docker（推荐）

```bash
mkdir go-drive-data
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  --restart unless-stopped \
  devld/go-drive
```

`/app/data` 包含数据库、本地盘、会话、缩略图缓存、临时文件和已安装的脚本 Drive。必须将它映射到容器外，否则删除容器时会丢失数据。

官方镜像包含 libvips 和 ffmpeg，并自动启用高性能图片、视频帧和音频封面缩略图处理器。

### 自定义配置

先从临时容器提取当前镜像自带的配置：

```bash
cid=$(docker create devld/go-drive)
docker cp "$cid:/app/config.yml" ./config.yml
docker rm "$cid"
```

然后挂载配置文件：

```bash
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  -v "$(pwd)/config.yml:/app/config.yml:ro" \
  --restart unless-stopped \
  devld/go-drive
```

## Docker Compose

```yaml
services:
  go-drive:
    image: devld/go-drive
    container_name: go-drive
    ports:
      - "8089:8089"
    volumes:
      - ./go-drive-data:/app/data
      # 需要自定义配置时取消下一行注释
      # - ./config.yml:/app/config.yml:ro
    restart: unless-stopped
```

```bash
docker compose up -d
```

## 预编译包

从 [GitHub Releases](https://github.com/devld/go-drive/releases) 下载对应平台的压缩包并解压：

```bash
# Linux
./go-drive

# 指定配置
./go-drive -c /path/to/config.yml
```

Windows 运行 `go-drive.exe`。应用默认读取工作目录中的 `config.yml`；没有配置文件时使用内置默认值并在 `:8089` 监听。

## 从源码构建

需要：

- `go.mod` 指定的 Go 版本（当前为 Go 1.26.4）。
- Node.js 24 和 npm。
- GNU Make。
- C 编译工具链；SQLite 依赖 CGO。

```bash
git clone https://github.com/devld/go-drive.git
cd go-drive
BUILD_VERSION=dev make all
```

`make all` 会构建前端、Monaco Editor、后端和发布压缩包。Web UI 与 i18n 资源会嵌入发布二进制，产物位于 `build/`。

仅开发前端时：

```bash
cd web
npm install
npm run dev
npm run lint
npm run build-web
```

## 首次登录

打开 `http://localhost:8089`，使用：

- 用户名：`admin`
- 密码：`123456`

登录后立即修改密码。公开部署前还应配置 HTTPS、可信代理、最小权限和备份，见[安全指南](../configuration/security.html)。

## 下一步

- [配置文件](../configuration/)
- [添加 Drive](../drives/)
- [用户、组和权限](../administration/access-control.html)
- [升级、备份与恢复](./upgrade-backup.html)

---
title: 文件桶
description: 使用 go-drive 文件桶发布受控上传下载入口，并配置访问 Token、文件类型限制和防盗链。
lang: zh-CN
translation_key: file-buckets
source_hash: cc899e1aad91f0785217a8c38c03854e739f5578eab5b9ae0302a08efd1ea98a
---

# 文件桶

文件桶提供一个面向程序的上传和公开读取入口。它不是 S3 协议，也不会使用普通用户的登录 Token。常见用途是搭建自托管图床，为博客或笔记工具提供图片上传服务。

在“管理员 → 文件桶”创建桶。

## 字段

| 字段 | 说明 |
| --- | --- |
| 名称 | URL 中的唯一桶名 |
| 目标路径 | 文件实际写入的虚拟目录，不以 `/` 开头 |
| 文件路径模板 | 服务器生成对象路径的模板 |
| 上传密钥 | 上传请求查询参数 `t`，应使用不可猜测值 |
| 下载 URL 模板 | 上传成功后返回的 URL |
| 允许自定义路径 | 允许调用方在 URL 中指定路径 |
| 最大大小 | 带单位的大小；不大于零表示不限制 |
| 允许类型 | MIME、扩展名或通配 MIME 列表 |
| 允许 Referer | 允许下载的 Referer 主机列表 |
| 缓存时间 | 下载响应 `Cache-Control` 的 max-age |

## 上传

```bash
# multipart/form-data，字段必须叫 file
curl -f -F 'file=@photo.jpg' \
  'https://drive.example.com/f/assets?t=UPLOAD_TOKEN'

# 原始文件流
curl -f -X POST --data-binary '@photo.jpg' \
  'https://drive.example.com/f/assets?t=UPLOAD_TOKEN'
```

开启自定义路径后：

```bash
curl -f -F 'file=@photo.jpg' \
  'https://drive.example.com/f/assets/custom/2026/photo.jpg?t=UPLOAD_TOKEN'
```

响应正文是下载 URL，并包含 `X-File-Size` 和 `X-File-Mime`。服务器会检测实际 MIME；不要只依赖客户端文件名。

## 路径模板

支持：`{year}`、`{month}`、`{date}`、`{hour}`、`{minute}`、`{second}`、`{millisecond}`、`{timestamp}`、`{rand}`、`{name}`、`{ext}`。

默认：

```text
{year}{month}{date}/{name}-{rand}{ext}
```

## 下载 URL 模板

支持 `{origin}`、`{bucket}`、`{key}`，默认：

```text
{origin}/f/{bucket}/{key}
```

下载接口是 `GET`/`HEAD /f/<桶名>/<路径>`。默认缓存时间为 `1d`；设置不大于零会返回 `no-cache`。

## 允许类型

允许类型可以混合使用：

```text
image/*,application/pdf,.jpg,.png
```

支持 MIME 类型、文件扩展名和通配 MIME（如 `image/*`）。

## 防盗链

"允许 Referer"字段限制哪些站点可以下载桶中的文件。填写主机名，逗号分隔，支持通配子域名。空条目允许没有 `Referer` 头的请求。

示例：

```text
example.com,*.example.com,cdn.other.com,
```

末尾的逗号产生一个空条目，允许不带 `Referer` 头的请求（如浏览器直接访问）。如果不加空条目，则只有来自列出域名的请求会被放行。

`Referer` 可以被客户端伪造，只适合减少普通盗链，不是认证机制。

## 示例：搭建图床

为博客或 Markdown 编辑器搭建一个简单图床的配置：

| 字段 | 值 |
| --- | --- |
| 名称 | `images` |
| 目标路径 | `blog/images` |
| 文件路径模板 | `{year}{month}/{name}-{rand}{ext}` |
| 上传密钥 | *（一个随机字符串）* |
| 允许类型 | `image/*` |
| 最大大小 | `10m` |
| 缓存时间 | `30d` |

上传图片：

```bash
curl -f -F 'file=@screenshot.png' \
  'https://drive.example.com/f/images?t=YOUR_TOKEN'
```

响应正文即为公开访问的 URL，可以直接用在 Markdown 中：

```markdown
![screenshot](https://drive.example.com/f/images/202607/screenshot-a1b2c3d4e5f6.png)
```

许多 Markdown 编辑器（Typora、PicGo 等）支持通过自定义命令或 API 上传图片，只需将上传地址指向桶的上传 URL 并带上上传密钥即可。

## 安全建议

- 为每个用途使用独立、随机的上传 Token，并定期轮换。
- 限制目标目录、文件大小和 MIME。
- 不需要自定义路径时保持关闭，避免调用方覆盖可预测路径。
- 文件桶读取是公开接口；敏感文件不要依赖 Referer 保护。
- 如果桶用于不可变静态资源，可配置长缓存；可覆盖资源应使用短缓存或版本化文件名。

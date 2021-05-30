---
title: Go-drive

---

- [开始使用](#开始使用)
  - [Docker](#docker)
  - [直接运行](#直接运行)
- [功能介绍](#功能介绍)
  - [添加存储映射](#添加存储映射)
  - [权限相关](#权限相关)
  - [挂载](#挂载)
  - [启动参数](#启动参数)
  - [配置文件](#配置文件)

## 开始使用

### Docker

```bash
mkdir go-drive-data
docker run -d --name go-drive -p 8089:8089 -v `pwd`/go-drive-data:/app/data devld/go-drive:0.4.0
```

其中 `go-drive-data` 是数据目录，为了保证数据在应用升级后保留，最好将其映射至容器外。

### 直接运行

在 [Release](https://github.com/devld/go-drive/releases) 页面下载对应平台的压缩包，解压后运行

- Linux

```
./go-drive
```

- Windows

直接运行 `go-drive.exe` 即可



当应用运行后，访问 `http://<你的 ip>:8089` 即可开始使用，默认的管理员用户为 `admin`，密码为 `123456`。



## 功能介绍

使用 Go-drive 可以将多种存储“挂载”在根路径下，方便地进行访问及管理，同时支持基于用户/组的权限控制。也可以将某一路径像 Linux 中的 `mount` 一样挂载在另一路径下。



### 添加存储映射

在 `管理员` -> `盘` 中，即可添加存储映射。并请注意，**任何对存储映射的修改均需点击“重新加载盘”才可生效**。



目前 Go-drive 支持以下六种后端存储：

- 本地文件

  映射目录至服务器本地的某个目录。默认情况下，`本地文件`的`根目录`为相对路径，并且被限制在 `<数据目录>/local` 目录中，如果要添加本地文件的映射，则需提前在 `<数据目录>/local` 下新建名为映射名称的目录。

  如：数据目录为 `/home/me/go-drive/data`，则新建名为 `test` 的本地文件映射，需要提前新建 `/home/me/go-drive/data/local/test` 目录。

  > 如果你希望去除上述的路径限制，可在启动参数添加 `-f`，禁用该限制。
  >
  > 但请注意，这将导致管理员用户可随意访问整个系统的文件。

- FTP

  映射 FTP 文件

- WebDAV 协议

  映射 WebDAV 协议的存储

- S3 兼容的云存储

  映射兼容 S3 协议的各种云存储，包括但不限于 `AWS S3`, `腾讯 COS`, `阿里云 OSS`。

  S3 的下载上传可不经过服务器，节省服务器带宽，可通过`上传代理`、`下载代理` 分别启用。

- OneDrive

  映射 OneDrive 存储。目前只在作者的个人微软账号进行过测试，不保证可以用于其他账号类型。

  OneDrive 的下载上传可不经过服务器，节省服务器带宽，可通过`上传代理`、`下载代理` 分别启用。

- Google Drive

  映射 Google Drive 存储。由于 Google Drive 没有 API 层面的路径结构，并且可在“目录”中存在同名文件，所以当 Go-drive 遇到同名文件时，会在文件名后面加上 `-<文件 id 前 6 位>`。

  Google Drive 中的某些文件类型在下载时会被导出至对应的文件，下面是对应的关系：

  - `application/vnd.google-apps.document`: 文档，转换为 `docx`
  - `application/vnd.google-apps.spreadsheet`：表格，转换为 `xlsx`
  - `application/vnd.google-apps.presentation`：演示，转换为 `pptx`
  - `application/vnd.google-apps.drawing`：Drawing，转换为 `svg`
  - `application/vnd.google-apps.script`：Script，转换为 `json`

  

### 权限相关

在 `管理员` -> `其他` 中，可以配置根目录的权限，当某个路径没有匹配到的权限规则时，将向父目录匹配，直至匹配到根路径。



在文件条目上鼠标右键（移动端长按）可呼出上下文菜单，在 `权限` 菜单项中可配置针对该条目的权限。

权限的匹配规则为：

- 越具体的路径优先级越高，即针对 `a/b/c` 的权限的优先级高于对于 `a/b` 的

- `拒绝(Rejected)` 的权限优先级高于 `接受(Accepted)`

- 越具体的`主体`优先级越高，即针对`用户 a` 的权限配置优先级高于针对 `组 b` 的，针对 `组 b` 的权限配置优先级高于 `ANY(针对任意用户，包括未登录的用户)`

  

### 挂载

要挂载某个条目到另一路径下，可在上下文菜单中选择 `挂载到`，并选择目的地。被挂载的条目的名称前面将会出现 `@` 符号。

> 权限仅针对路径，如果将某个条目挂载到另一位置，则该条目在挂载位置的权限将只会匹配挂载位置的权限。



### 启动参数

- `-c` 指定配置文件，如果未指定，默认会尝试 `config.yml`
- `-v` 显示版本信息
- `-h` 显示帮助信息
- `-show-config` 显示解析到的配置文件



### 配置文件

```yaml
# 监听地址及端口
# 默认为 `:8089`，即在所有接口(`0.0.0.0`)上监听 `8089` 端口。如果要监听某个特定的接口，则可传入 `<接口 ip>:<端口号>`
listen: :8089

# 系统名称/标题，默认为 "Drive"
# 请注意：由于前端的 PWA 缓存问题，此参数可能不会立即生效
app-name: Drive

# 数据目录
# Go-drive 所有的数据均在该目录下，如果使用 Docker 等容器运行，则需将该路径映射至容器外，否则会导致数据丢失。
# - `lang` 国际化文本
# - `local` 当 `-f` 参数未启用时，`本地文件` 的映射将始终被限制在该目录中
# - `sessions` 用户会话
#- `temp` 临时目录
# - `thumbnails` 文件的缩略图缓存
# - `upload_temp` 分片上传的临时文件
# - `data.db` SQLite 数据库文件
data-dir: ./

# 静态文件路径
# 通常为前端资源文件位置，默认为 `./web`
web-dir: ./web

# 国际化文件位置
# 默认为 `./lang`
lang-dir: ./lang
# 默认语言
# 默认为 `en-US`，当用户浏览器的语言不受支持时，将回退到该语言
default-lang: en-US

# 当前端要求代理某文件的下载时，所支持的最大的文件大小
# 默认为 `1048576` ，即 1MB
proxy-max-size: 1048576 # 1M

# 并发任务数
# 默认为 `100`，为复制、移动、删除等异步任务的并发数
max-concurrent-task: 100

# 禁用`本地文件` 映射的路径限制
free-fs: false

thumbnail:
  # 缩略图缓存有效期
  # 默认为 `720h`，即 30 天。当文件发生变化时（通过文件的上次修改时间和大小决定），缓存也会失效
  ttl: 720h
  #生成缩略图的并发数
  # 默认为 (CPU 数量 / 2)，目前图片的缩略图生成比较耗性能和内存。
  #concurrent: 4

  # 缩略图生成器。 目前支持三种类型: image, text, shell
  # file-types 指这个生成器支持的文件扩展名
  handlers:
    # image: 内嵌的图片缩略图生成(只支持 jpg, png, gif)
    # 这个目前性能不佳，不推荐使用
    - type: image
      file-types: jpg,jpeg,png,gif
      #config:
      #  # 最大支持的文件大小
      #  max-size: 33554432 # 32MB
      #  # 最大支持的图片大小 (宽 * 高)
      #  max-pixels: 36000000 # 6000*6000
      #  # 生成的缩略图宽度(像素)
      #  size: 220
      #  # 缩略图图片质量，1 ~ 100，越大质量越好
      #  quality: 50

    # text: 内嵌的针对文本文件的生成器
    # 读取文件文件的部分内容来生成一个 svg 图片
    - type: text
      file-types: txt,md,xml,html,css,scss,js,json,jsx,properties,yml,yaml,ini,c,h,cpp,go,java,kt,gradle,ps1
      #config:
      #  font-size: 12
      #  # 生成的缩略图宽度(像素)
      #  size: 220
      #  # 最多读取的文件长度
      #  max-read: 8192
      #  # 生成的图片的 padding
      #  padding: 10

    # shell: 通过执行外部命令来生成缩略图，比如 ffmpeg
    # 文件的内容会被写入标准输入(stdin)
    # 生成的缩略图应该写出到标准输出(stdout)
    # 如果命令返回非 0 状态，表示生成失败
    # 一些相关的环境变量会被设置:
    #
    # GO_DRIVE_ENTRY_TYPE: file|dir
    # GO_DRIVE_ENTRY_PATH: 引号括起来的文件路径(不是本地文件系统路径)
    # GO_DRIVE_ENTRY_NAME: 引号括起来的文件名
    # GO_DRIVE_ENTRY_SIZE: 文件大小
    # GO_DRIVE_ENTRY_MOD_TIME: 文件修改时间，毫秒时间戳
    # GO_DRIVE_ENTRY_READABLE: true|false 这个文件是否可读
    #- type: shell
    #  file-types: mp4,avi
    #  config:
    #    # 生成缩略图的命令
    #    # 比如，下面的命令调用 ffmpeg 为视频生成缩略图
    #    shell: ffmpeg.exe -hide_banner -loglevel error -i - -frames:v 1 -vf scale=220:-1 -f mjpeg -
    #    # 输出的缩略图的 mime type
    #    mime-type: image/jpeg
    #    # 输出的文件名
    #    filename: image.jpg
    #    # 如果设置为 false，那么文件内容不会写入到 stdin
    #    write-content: true
    #    # 最大支持的文件大小，如果 <= 0，则没有限制
    #    max-size: -1
    #    # 生成缩略图的超时时间，如果 <= 0, 则没有限制，默认为永不超时
    #    timeout: 10m

auth:
  # 用户 Session Token 有效时间
  # 默认为 `2h`，两小时
  validity: 2h
  # 当用户与系统交互时，自动刷新 Token 有效期
  auto-refresh: true

# API base 路径
# 传递给前端的参数，通常情况下，不需要指定
# 当 go-drive 在反向代理(如 Nginx)后面且在子路径下时，需要指定
# 请注意：由于前端的 PWA 缓存问题，此参数可能不会立即生效
api-path: ""

# OAuth 认证时的重定向 URL
oauth-redirect-uri: https://go-drive.top/oauth_callback
```



> `https://go-drive.top/oauth_callback`，该网页是一个静态网页，没有与任何后端交互，源码在 [https://github.com/devld/go-drive/blob/gh-pages/oauth_callback.html](https://github.com/devld/go-drive/blob/gh-pages/oauth_callback.html)

# go-drive

[English version](#features)

Demo: [https://demo.go-drive.top](https://demo.go-drive.top)

Doc: [https://go-drive.top](https://go-drive.top)

## 功能

- 文件管理，拖拽/粘贴上传，拖拽管理文件
- 文件打包下载
- 基于用户/组的权限控制
- 图片浏览([PhotoSwipe](https://github.com/dimsemenov/PhotoSwipe))
- 音乐播放([APlayer](https://github.com/DIYgod/APlayer))
- 文本编辑([CodeMirror](https://github.com/codemirror/))
- 代码编辑([Monaco Editor](https://github.com/microsoft/monaco-editor))
- 展示缩略图
- 路径挂载
- Drive 管理界面
- 文件搜索
- 通过 WebDAV 访问
- 定时任务([gocron](https://github.com/go-co-op/gocron))

## 目前支持的 Drives

- 本地文件
- FTP
- SFTP
- WebDAV 协议
- S3 兼容的云存储
- OneDrive
- Google Drive
- Dropbox(JavaScript)
- 七牛云(JavaScript)

## 如何使用

### Docker

```shell
mkdir go-drive-data
docker run -d --name go-drive -p 8089:8089 -v `pwd`/go-drive-data:/app/data devld/go-drive
```

其中 `go-drive-data` 是数据目录，为了保证数据在应用升级后保留，最好将其映射至容器外。

### 直接运行

在 [Release](https://github.com/devld/go-drive/releases) 页面下载对应平台的压缩包，解压后运行

- Linux

```shell
./go-drive
```
- Windows

直接运行 `go-drive.exe` 即可

> 默认用户为 `admin`，密码 `123456`

## Features

- files management, drag-and-drop/paste upload, drag-and-drop file management
- Zip package download
- User/group-based permission control
- Image gallery([PhotoSwipe](https://github.com/dimsemenov/PhotoSwipe))
- Music player([APlayer](https://github.com/DIYgod/APlayer))
- Text Editor([CodeMirror](https://github.com/codemirror/))
- Code Editor([Monaco Editor](https://github.com/microsoft/monaco-editor))
- Thumbnails
- Path mounting
- Drive management
- Files searching
- Access via WebDAV 
- Scheduled Jobs([gocron](https://github.com/go-co-op/gocron))

## Currently supported drives

- Local
- FTP
- SFTP
- WebDAV
- S3
- OneDrive
- Google Drive
- Dropbox(JavaScript)
- Qiniu(JavaScript)

## How to use

### Docker

```shell
mkdir go-drive-data
docker run -d --name go-drive -p 8089:8089 -v `pwd`/go-drive-data:/app/data devld/go-drive
```

`go-drive-data` is the data directory. To ensure that the data is retained after the application is upgraded, it is best to map it outside the container.

### Release

Download binary file from the [Release](https://github.com/devld/go-drive/releases) page, decompress and run

- Linux

```shell
./go-drive
```
- Windows

Just run `go-drive.exe`

> Default user is `admin`, its password is `123456`

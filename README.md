# go-drive

[English version](#features)

Demo: [https://demo.go-drive.top](https://demo.go-drive.top)

Doc: [https://go-drive.top](https://go-drive.top)

## 功能

- 基本的文件管理(上传，下载，复制，移动，重命名，删除)
- 基于用户/组的访问控制
- 图片浏览
- 文本编辑
- 路径挂载
- 在 Drive 之间复制文件(夹)
- Drive 管理界面

## 目前支持的 Drives

- 本地文件
- FTP
- WebDAV 协议
- S3 兼容的云存储
- OneDrive
- Google Drive

## 如何使用

### Docker

```shell
mkdir go-drive-data
docker run -d --name go-drive -p 8089:8089 -v `pwd`/go-drive-data:/app/data devld/go-drive:0.3.0
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

默认用户为 `admin`，密码 `123456`

## Features

- Basic file management(upload, download, copy, move, rename, delete)
- User/group-based access control
- Images gallery
- Text file editing
- Path mounting
- Copy files/folders across drives
- Drive-mapping management

## Currently supported drives

- Local
- FTP
- WebDAV
- S3
- OneDrive
- Google Drive

## How to use

### Docker

```shell
mkdir go-drive-data
docker run -d --name go-drive -p 8089:8089 -v `pwd`/go-drive-data:/app/data devld/go-drive:0.3.0
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

Default user is `admin`, its password is `123456`


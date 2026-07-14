---
title: Google Drive
description: 使用 OAuth 凭据、重定向 URI 和 API Scope 将 Google Drive 或共享云端硬盘连接到 go-drive。
lang: zh-CN
translation_key: drive-google-drive
source_hash: 86c052b42dae64b8c273e2a43bc12b8d123d77b7d5b80d0f3c32048936f232f5
---

# Google Drive

## 创建 OAuth 应用

1. 在 [Google Cloud Console](https://console.cloud.google.com/) 创建项目。
2. 启用 Google Drive API。
3. 配置 OAuth 同意屏幕。
4. 添加作用域：
   - `https://www.googleapis.com/auth/drive`
   - `https://www.googleapis.com/auth/userinfo.profile`
5. 创建“Web 应用”OAuth 客户端。
6. 添加重定向 URI，默认是 `https://go-drive.top/oauth_callback`。

如果使用测试状态的外部应用，refresh token 可能受 Google 测试应用政策限制。长期运行前应按账号类型和组织政策正确发布应用。

可使用自己的回调页：

```yaml
oauth-redirect-uri: https://drive.example.com/oauth_callback
```

Google 控制台中的 URI 必须与配置完全一致。

## 添加 Drive

填写 Client ID、Client Secret、缓存 TTL 和“代理缩略图”，完成 OAuth 后选择个人盘或共享盘，保存并重新加载 Drive。

- 缓存 TTL 默认 `4h`；外部直接修改文件后可清除缓存。
- 代理缩略图默认开启，适合不能让浏览器直接访问 Google 缩略图的环境。

Google Drive API 没有传统路径模型，并允许同一目录存在同名文件。go-drive 遇到同名条目时会给名称追加文件 ID 的前 6 位。

## Google 原生文件导出

| Google 类型 | 下载格式 |
| --- | --- |
| Docs 文档 | `.docx` |
| Sheets 表格 | `.xlsx` |
| Slides 演示 | `.pptx` |
| Drawing | `.svg` |
| Apps Script | `.json` |

Google Drive 不支持原生复制文件夹，目录复制会由 go-drive 递归执行。

> Google Cloud Console 界面会调整，但作用域和 go-drive 字段以上述当前实现为准。

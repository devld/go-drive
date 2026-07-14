---
title: 脚本 Drive 开发与安装
description: 安装第三方脚本 Drive，或使用 JavaScript 开发 go-drive 存储适配器和浏览器直传集成。
lang: zh-CN
translation_key: script-drives
source_hash: ce820c46430f71ae9d00edd19ff2020dbd7bacd12c579a45faea35ac60c1e32f
---

# 脚本 Drive 开发与安装

JavaScript Drive 可以在不重新编译 go-drive 的情况下添加存储后端。Dropbox、七牛云等扩展就是这种类型。

## 安装

进入“管理员 → 其他盘”，刷新仓库后选择扩展安装。一个扩展通常包含：

- `<name>.js`：服务器端 Drive 实现。
- `<name>-uploader.js`：可选的浏览器上传适配器。

默认仓库：

```text
https://api.github.com/repos/devld/go-drive/contents/script-drives
```

可在配置中修改：

```yaml
drives-dir: script-drives
drive-uploaders-dir: drive-uploaders
drive-repository-url: https://example.com/my-drives.json
```

自定义仓库返回 GitHub Contents API 风格数组，至少包含 `name` 和 `download_url`：

```json
[
  { "name": "example.js", "download_url": "https://example.com/example.js" },
  { "name": "example-uploader.js", "download_url": "https://example.com/example-uploader.js" }
]
```

安装后在 Drive 管理页创建对应类型并重新加载。

## 开发入口

从当前模板开始：

- [`docs/script-drive-template.js`](https://github.com/devld/go-drive/blob/master/docs/script-drive-template.js)
- [`docs/scripts/global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts)
- [`docs/scripts/drive.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/drive.d.ts)
- [`docs/drive-uploaders`](https://github.com/devld/go-drive/tree/master/docs/drive-uploaders)

模板通过 TypeScript reference 提供编辑器补全，但运行时仍是服务器端 JavaScript。实现应：

- 定义唯一类型名、显示名、说明和配置表单。
- 正确实现 Get/List/Save/MakeDir/Copy/Move/Delete/Content。
- 对不支持的原生操作返回 Unsupported，让调度层使用通用回退。
- 使用传入的 context，并在可中止操作中传播取消。
- 及时关闭响应体、reader 和远端连接。
- 不把 Token、密码或签名 URL 输出到日志。

## 浏览器上传器

上传器在用户浏览器中运行，可用于 S3 类直传。它必须处理 CORS、进度、取消、错误和服务端返回结果。服务器脚本与上传器是两个不同的信任面，审核扩展时必须同时检查。

## 调试

1. 在测试实例中安装或编辑脚本。
2. 使用专门测试账号和目录。
3. 开启 `GO_DRIVE_DEBUG=1` 获取临时调试信息。
4. 分别测试空文件、大文件、覆盖、目录、取消、网络错误和凭据过期。
5. 测试完成后关闭 debug，并重新加载 Drive。

脚本 VM 会为调用隔离运行状态，但脚本仍可以访问网络和 Drive 数据；不要把它当作不可信代码沙箱。

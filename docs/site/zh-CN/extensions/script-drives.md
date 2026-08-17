---
title: 脚本 Drive 开发与安装
description: 安装第三方脚本 Drive，或使用 JavaScript 开发 go-drive 存储适配器和浏览器直传集成。
lang: zh-CN
translation_key: script-drives
source_hash: 6a7b86293bc490ca4a6f87758a3860d865fe3b4f16e752b3d4950ae8e77a4645
---

# 脚本 Drive 开发与安装

JavaScript Drive 可以在不重新编译 go-drive 的情况下添加存储后端。Dropbox、七牛云等扩展就是这种类型。

它适用于能通过 HTTP/HTTPS API 把文件和目录映射成路径的存储服务，并不是通用协议运行时。SMB/Samba、SFTP、FTP、本地文件系统，以及依赖原始 socket、原生库、Node.js 包或操作系统命令的服务，应实现内置 Go Drive。

## 安装

进入“管理员 → 其他盘”，刷新仓库后选择扩展安装。一个扩展通常包含：

- `<name>.js`：服务器端 Drive 实现。
- 可选的浏览器上传适配器，由 `@uploader` 声明，安装时复制到 `drive-uploaders/<name>.js`。

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

自定义仓库返回 GitHub Contents API 风格数组，包含 `name` 和 `download_url`。只会下载 `.js` 条目：

```json
[
  { "name": "example.js", "download_url": "https://example.com/example.js" },
  { "name": "example-uploader.js", "download_url": "https://example.com/example-uploader.js" }
]
```

服务器端脚本在开头注释中声明元数据：

```js
// @name Example Cloud
// @version 1.0.0
// @uploader example-uploader.js
// @description Example Cloud REST API adapter.
```

刷新仓库会作为后台任务运行。go-drive 会下载 listing 中的每个 `.js`，然后只保留同时声明了 `@name` 和 `@version` 的脚本。上传器仅在 Drive 脚本通过 `@uploader` 引用时保留。文件先写入 `script-drives/.repo/`，安装/更新时再复制到 `script-drives/` 和 `drive-uploaders/`。管理页会根据版本号提供更新按钮，无需先卸载脚本 Drive。如果上传器发生变化，也需要同步提升服务器端脚本版本号。已安装但缺少 `@name` 或 `@version` 的脚本不会出现在列表中。

安装后在 Drive 管理页创建对应类型并重新加载。

## 开发入口

从当前模板开始：

- [`docs/script-drive-template.js`](https://github.com/devld/go-drive/blob/master/docs/script-drive-template.js)
- [`docs/scripts/global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts)
- [`docs/scripts/env/drive.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/env/drive.d.ts)
- [`docs/drive-uploaders`](https://github.com/devld/go-drive/tree/master/docs/drive-uploaders)
- [`script-drives/AGENTS.md`](https://github.com/devld/go-drive/blob/master/script-drives/AGENTS.md)：供 AI Agent 和开发者使用的完整适用性判断、实现契约、API 清单与端到端示例。

模板通过 TypeScript reference 提供编辑器补全，但运行时仍是服务器端 JavaScript。实现应：

- 定义唯一类型名、显示名、说明和配置表单。
- `configForm` 只用于静态配置；多步动态配置应按照原生 Drive 生命周期实现显式的 `initConfig(ctx, config, utils)` 和 `init(ctx, data, config, utils)` 回调。以下划线开头的表单字段名是保留字段。
- 通过 `utils.OAuthInitConfig`、`utils.OAuthInit` 和 `utils.OAuthGet` 显式处理 OAuth，不再提供自动 OAuth 回调。
- 实现 `createInstance(config, utils)`，通过 `utils.Data.Load("key", ...)` 按需读取 Drive 所需的动态初始化字段。
- 用 `defineDrive` 实现 `get`、`list`，以及 `getURL` 或 `getReader`，再按服务能力实现写入、上传、下载和缩略图方法。
- 原生 `copy` 不可用时返回 Unsupported，调度层会流式复制；`move` 没有 copy-and-delete 回退。
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

调用由并发 VM 池执行，普通可变全局变量不是可靠的共享状态；只有可 JSON 序列化的 `$` 实例属性会在 VM 间同步。脚本仍可以访问网络和 Drive 数据；不要把它当作不可信代码沙箱。

---
title: 通过 WOPI 编辑 Office 文档
description: 将 go-drive 连接到自托管 Collabora Online，在浏览器中查看和编辑 Office 文档。
lang: zh-CN
translation_key: wopi
source_hash: 344670fca042f98d0a129647b1889e67bbbb9ab09f7be82f5d19a392bf0591da
---

# 通过 WOPI 编辑 Office 文档

go-drive 实现了查看和编辑文档所需的 WOPI Host 接口，可连接提供标准 discovery 的 Office 服务。当前首个受支持的部署目标是自托管 Collabora Online。

## 启用 WOPI

配置 Office 服务暴露的 discovery 文档：

```yaml
wopi:
  discovery-url: https://office.example.com/hosting/discovery
```

重启 go-drive。Web 界面会从 discovery 获取支持的扩展名和 `view`/`edit` action，并为匹配文件显示“使用 Office 打开”。未配置或留空 `discovery-url` 时关闭 WOPI。

go-drive 进程必须能访问 discovery 地址；反过来，Office 服务也必须能访问用户打开文档时使用的每个 go-drive 公网域名。

## 反向代理和多域名

go-drive 根据浏览器当前 `Origin` 生成 WOPISrc，因此同一实例可以通过多个域名访问，不需要固定的公网 URL 配置。每个公网域名都需要：

- 生产环境使用 HTTPS；
- 反向代理保留原始 `Host` 请求头；
- 将 `api-path/wopi/*` 转发到同一个 go-drive 实例；
- 在 Collabora 的 `alias_groups` 中允许对应 host 或 alias。

浏览器 `Origin` 必须与请求 `Host` 一致。如果代理把 `Host` 改写成内部服务名，创建编辑会话会失败。

子路径部署仍按正常方式配置 `api-path`。例如 `api-path: /drive` 会把 WOPI 接口放在 `/drive/wopi/` 下。

## 身份认证和权限

只有已登录用户才能打开 Office Handler。go-drive 会针对单个用户和单个文件签发独立的随机 WOPI token，普通登录 token 不会发送给 Office 服务。每次 WOPI 回调都会重新加载用户，并经过正常的用户/组根路径、路径权限和路径属性包装层。

WOPI 会话在 10 小时后过期。token 和锁只保存在当前进程中；重启 go-drive 会使已打开的编辑器失效，用户需要重新打开文档。

## 锁范围和外部修改

WOPI 锁只协调 WOPI 客户端，不会阻止：

- WebDAV 写入；
- 普通 Web UI 上传或文本编辑；
- 自动任务；
- 直接修改底层第三方存储中的文件。

创建 WOPI 锁时，go-drive 会根据底层条目的路径、修改时间和大小记录一个版本。WOPI 保存前如果这些值发生变化，`PutFile` 会返回冲突，而不是静默覆盖外部修改。这只是尽力检测：如果后端不能提供可靠的修改时间，就无法完整发现冲突。

锁在 30 分钟内没有刷新就会过期。多个 go-drive 进程之间不会共享会话和锁；启用 WOPI 编辑时应使用单个应用实例。

## 兼容性边界

当前实现包括 CheckFileInfo、GetFile、PutFile、PutRelativeFile、Lock、GetLock、RefreshLock、Unlock 和 UnlockAndRelock，暂未校验 Microsoft proof key。Microsoft 365 for the web 还要求加入 Cloud Storage Partner Program、登记域名并提供更严格的全局冲突处理；此版本应使用 Collabora Online 作为受支持的部署目标。

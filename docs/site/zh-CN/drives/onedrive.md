---
title: OneDrive
description: 使用正确的 OAuth 和租户设置，将个人版、组织版、世纪互联或 SharePoint OneDrive 连接到 go-drive。
lang: zh-CN
translation_key: drive-onedrive
source_hash: 32877155616e5962830b831342513c071674974370358ccfce07444ba346b869
---

# OneDrive

OneDrive Drive 支持 Microsoft 全球版、世纪互联版、个人/组织账号以及 SharePoint 站点。

## 注册应用

在对应的 Microsoft Entra 管理门户注册 Web 应用：

- 全球版：<https://portal.azure.com/>
- 世纪互联版：<https://portal.azure.cn/>

创建客户端密钥，并添加 Web 重定向 URI。默认值是：

```text
https://go-drive.top/oauth_callback
```

也可以在 `config.yml` 中设置自己的回调页：

```yaml
oauth-redirect-uri: https://drive.example.com/oauth_callback
```

门户中的 URI 必须与配置完全一致。

## 权限

普通个人盘/组织盘需要委托权限：

- `User.Read`
- `Files.ReadWrite`
- `offline_access`

访问 SharePoint 站点时使用 `Files.ReadWrite.All` 代替 `Files.ReadWrite`，并按组织策略完成管理员同意。

## go-drive 字段

| 字段 | 说明 |
| --- | --- |
| 区域 | 全球版选择 `global`，世纪互联选择 `china` |
| Tenant | `common`、`organizations` 或 `consumers`，需与应用账号类型一致 |
| Client ID | 应用程序（客户端）ID |
| Client Secret | 客户端密钥的值，不是密钥 ID |
| SharePoint 站点 | 可选，例如 `https://example.sharepoint.com/sites/team` |
| 代理上传/下载 | 强制流量通过 go-drive |
| 缓存 TTL | 目录项缓存时间；不大于零关闭缓存 |

世纪互联通常使用 `common` tenant。保存后按界面提示完成 OAuth，再选择要映射的 Drive 或 SharePoint 站点，最后重新加载 Drive。

客户端密钥会过期；到期前在门户创建新密钥并更新 Drive。修改 SharePoint 或盘选择后也应重新加载并清理旧缓存。

> Microsoft 门户界面会调整，但权限名称和 go-drive 字段以上述当前实现为准。

---
title: 配置文件参考
description: 查阅 go-drive 的网络、数据库、存储、搜索、WebDAV、缩略图、自动任务和安全配置选项。
lang: zh-CN
translation_key: configuration
source_hash: 507e3efd917833d29527a148c1209165a1c9fb98ed3aafbe81cdd0cb6d79af89
---

# 配置文件参考

go-drive 使用 YAML 配置。默认读取当前目录的 `config.yml`，也可用 `-c` 指定。运行 `go-drive -show-config` 可查看合并默认值后的配置。

以下示例覆盖当前公开配置项；未使用的功能保持关闭或留空。

```yaml
listen: :8089

# 只有直接连接 go-drive 的反向代理才应加入此列表
# trusted-proxies:
#   - 127.0.0.1
#   - 172.16.0.0/12

db:
  type: sqlite               # sqlite 或 mysql
  name: data.db              # SQLite 文件名或 MySQL 数据库名
  # host: 127.0.0.1
  # port: 3306
  # user: go_drive
  # password: change-me
  # config:
  #   loc: Local

data-dir: ./data
temp-dir: ""                 # 空值表示 data-dir/temp

drives-dir: script-drives
drive-uploaders-dir: drive-uploaders
drive-repository-url: https://api.github.com/repos/devld/go-drive/contents/script-drives

oauth-redirect-uri: https://go-drive.top/oauth_callback
max-concurrent-task: 100
free-fs: false
signature-ttl: 12h

thumbnail:
  ttl: 720h
  # concurrent: 4            # 默认 max(CPU/2, 1)
  handlers:
    - type: image
      tags:
      file-types: jpg,jpeg,png,gif,webp
    - type: text
      tags:
      file-types: txt,md,xml,html,css,scss,js,json,jsx,properties,yml,yaml,ini,c,h,cpp,go,java,kt,gradle,ps1

auth:
  validity: 2h
  auto-refresh: true
  # providers: []

# web-dav:
#   enabled: true
#   prefix: /dav
#   allow-anonymous: false
#   max-cache-items: 1000

search:
  enabled: false
  type: sqlite

cache:
  type: mem
  clean-period: 10m

api-path: ""
web-path: ""
```

## 基础选项

| 选项 | 默认值 | 说明 |
| --- | --- | --- |
| `listen` | `:8089` | HTTP 监听地址 |
| `trusted-proxies` | 空 | 可以提供 `X-Forwarded-For` 的代理 IP/CIDR |
| `data-dir` | `./data` | 数据库、本地盘、脚本、会话、缓存等数据目录 |
| `temp-dir` | `data-dir/temp` | 上传、复制等临时文件目录 |
| `max-concurrent-task` | `100` | 复制、移动、删除等后台任务并发数 |
| `free-fs` | `false` | 是否允许本地 Drive 使用绝对路径；风险很高 |
| `signature-ttl` | `12h` | 文件内容和缩略图签名 URL 的有效时间 |
| `oauth-redirect-uri` | 项目回调页 | OneDrive/Google Drive OAuth 回调地址 |
| `api-path` | 空 | 反向代理子路径，例如 `/drive` |
| `web-path` | 空 | 静态资源路径覆盖；通常留空 |

旧版本的 `web-dir`、`lang-dir` 和 `default-lang` 已删除：发布二进制已经嵌入 Web UI 和语言资源。

## 数据库

SQLite 适合单实例部署。数据库位于 `data-dir/<db.name>`，默认启用 WAL 和 5 秒 busy timeout。

MySQL 至少需要设置 `type`、`host`、`name`、`user` 和 `password`。`db.config` 会作为 DSN 参数传给 GORM；不要把真实密码提交到版本库。

## LDAP

本地密码登录始终可用。添加 LDAP：

```yaml
auth:
  validity: 8h
  auto-refresh: true
  providers:
    - type: ldap
      config:
        url: ldaps://ldap.example.com:636
        start-tls: "false"
        skip-tls-verify: false
        bind-dn: cn=readonly,dc=example,dc=com
        bind-password: change-me
        base-dn: ou=users,dc=example,dc=com
        user-filter: "(uid=%s)"
        username-attr: uid
        group-base-dn: ou=groups,dc=example,dc=com
        group-filter: "(memberUid=%s)"
        group-name-attr: cn
        group-mapping:
          admin: ldap-admins
          staff: ldap-users,ldap-staff
```

- `%s` 替换为经过 LDAP 转义的用户名/UID，适合 `posixGroup`。
- `%d` 替换为用户完整 DN，适合 `groupOfNames` 或 AD，例如 `(member=%d)`。
- `group-mapping` 的键是 go-drive 组，值是一个或多个上游组。
- LDAP 用户首次成功登录时自动创建；启用组搜索后，每次登录同步组成员关系。
- 用户名精确且区分大小写；未知用户按 `providers` 配置顺序尝试认证。
- 已标记为外部来源的用户先尝试对应提供方；提供方失败时再尝试该用户的本地密码。JIT 创建的 LDAP 用户默认没有可猜测的本地密码。
- 不建议使用 `skip-tls-verify: true`。生产环境使用 LDAPS 或 StartTLS 并配置可信证书。

## 缩略图处理器

处理器类型为 `image`、`text` 或 `shell`。Shell 处理器支持 `shell`、`mime-type`、`write-content`、`max-size` 和 `timeout` 等配置，详见[预览与缩略图](../features/preview-thumbnail.html)。官方 Docker 镜像中的配置会启用 libvips/ffmpeg；从镜像提取配置可以获得对应模板。

## WebDAV、搜索和缓存

- WebDAV 默认关闭。`allow-anonymous` 仍受路径权限约束；公开启用前务必测试匿名权限。
- 搜索器当前为 `sqlite`，旧的 `bleve` 配置已经无效。
- `web-dav.max-cache-items` 控制 WebDAV 文件对象缓存上限。
- 全局 `cache` 当前使用内存实现，`clean-period` 控制定期清理周期。

更多内容：

- [反向代理](./reverse-proxy.html)
- [安全指南](./security.html)
- [搜索与索引](../features/search.html)
- [WebDAV](../features/webdav.html)

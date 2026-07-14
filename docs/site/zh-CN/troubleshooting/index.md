---
title: 故障排查
description: 排查 go-drive 启动、OAuth、存储、上传、WebDAV、反向代理、搜索、权限、缓存和缩略图问题。
lang: zh-CN
translation_key: troubleshooting
source_hash: 6b8300f5d0c229c78d54908572a7e8ae4fe3a2e5c2585995527e77b806f3391c
---

# 故障排查

## 保存 Drive 后没有生效

保存后点击“管理员 → 盘 → 重新加载盘”。如果加载失败，查看后端日志。编辑密码或 Secret 时保留界面提供的隐藏占位值，除非确实要替换密钥。

## 本地 Drive 提示路径不存在

- `free-fs: false`：填写相对路径，当前版本会自动创建 `<data-dir>/local/<路径>`；检查数据目录写权限。
- `free-fs: true`：填写进程或容器内已经存在的绝对路径；检查宿主机目录是否挂载进容器。

## 反向代理后所有用户显示同一 IP

确认代理发送 `X-Forwarded-For`，并把直接代理 IP/CIDR 加入 `trusted-proxies`。不要信任整个互联网。修改后重启。

## 登录返回 429

同一客户端 IP 在 5 分钟内失败 5 次会被暂时限制。修正密码或 LDAP 配置并等待窗口结束。如果所有用户一起被限制，通常是 `trusted-proxies` 配置错误，应用只看到了代理 IP。

## 大文件上传失败

- 检查 Nginx `client_max_body_size`、超时和 `proxy_request_buffering`。
- 检查 `temp-dir` 磁盘空间和权限。
- S3/OneDrive 直传检查浏览器控制台、CORS 和防盗链。
- 代理传输检查服务器出口带宽和云服务超时。
- 空文件失败时确认代理没有错误移除或改写请求体。

## S3 上传失败

- 后台请求失败：检查 endpoint、region、path-style、凭据和桶权限。
- 只有浏览器上传失败：检查 CORS、站点 origin 和 Referer 白名单。
- 内网环境无法访问预签名 URL：开启代理上传/下载。

## OAuth 失败

- 控制台重定向 URI 必须与 `oauth-redirect-uri` 完全一致。
- 检查 Client ID、Client Secret 的“值”和到期时间。
- OneDrive tenant、区域和账号类型必须匹配。
- Google 应用测试状态可能限制 refresh token 生命周期。
- 服务器与浏览器时钟应准确。

## 文件列表陈旧

外部系统修改文件后清除对应 Drive 缓存，或缩短/关闭 `cache_ttl`。搜索结果还需重新索引相关路径。

## 搜索没有结果

1. 确认 `search.enabled: true`、`type: sqlite` 并已重启。
2. 创建并等待索引任务完成。
3. 检查 `+`/`-` 过滤规则。
4. 用当前用户检查根路径和读取权限。
5. 外部修改后重新索引。

## WebDAV 连接失败

- URL 必须包含配置的完整 prefix 和尾部 `/`。
- 使用 go-drive 用户名和密码，而不是浏览器 Token。
- 检查 HTTPS 证书。
- 代理必须允许 WebDAV 方法。
- 子路径部署时 `api-path`、WebDAV prefix 和代理 location 要一致。
- 先用 curl/rclone 验证，再排查操作系统客户端限制。

## 缩略图不生成

- 检查扩展名是否在 handler 的 `file-types` 中。
- 检查路径映射 tag 是否存在于对应 handler。
- Shell handler 检查命令、`mime-type`、超时和程序是否安装。
- 远端文件使用 `write-content` 时确认可读权限和网络。
- 失败会被缓存；修复后重启可清除失败标记并重试。

## 任务不执行

- Cron 必须是标准 5 段，不能使用 Quartz `?` 或秒字段。
- 检查任务是否启用、进程时区和下一次运行时间。
- 文件事件仅支持 updated/deleted，检查路径模式。
- 查看执行历史和错误；先使用手动执行验证动作。
- 避免任务输出再次匹配自身事件触发器。

## 提交问题时

附上 `go-drive -v` 输出、部署方式、操作系统/架构、数据库和 Drive 类型、复现步骤及日志。配置中的密码、Token、Secret、签名 URL 和个人路径必须脱敏。

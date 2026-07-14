---
title: SFTP Drive
lang: zh-CN
translation_key: drive-sftp
source_hash: 77306da7eab2165c39fcda47c93733d3e4a4210f847f074b2620b41554a1a211
---

# SFTP Drive

| 字段 | 说明 | 默认值 |
| --- | --- | --- |
| 主机 | SSH/SFTP 主机名或 IP | 必填 |
| 端口 | SSH 端口 | `22` |
| 用户名 | SSH 用户 | 必填 |
| 密码 | 密码认证，可与私钥二选一 | 空 |
| 私钥 | PEM/OpenSSH 私钥内容 | 空 |
| 主机公钥 | SSH authorized-key 格式的固定主机密钥 | 空 |
| 根路径 | 映射的远端绝对路径 | `/` |
| 缓存 TTL | 目录项缓存时间；不大于零关闭缓存 | 关闭 |

生产环境应填写“主机公钥”以防中间人攻击。可在可信网络中使用 `ssh-keyscan` 获取候选值，但必须通过另一可信渠道核对指纹后再保存。

根路径必须以 `/` 开头。文件内容经过 go-drive；移动/重命名使用远端操作，复制由 go-drive 通用任务读取后重新上传。

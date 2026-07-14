---
title: 升级、备份与恢复
description: 安全升级 go-drive，并备份或恢复数据库、配置、本地文件、会话、脚本及生成的缓存数据。
lang: zh-CN
translation_key: upgrade-backup
source_hash: 98ce5aee55fb83b5e59d1fdb9bf700263cbde1c8e550b8b619d9d51f3c015bf5
---

# 升级、备份与恢复

## 升级前

1. 阅读目标版本的 release notes。
2. 停止写入或进入维护窗口。
3. 停止 go-drive，确保数据库和 WAL 已落盘。
4. 备份完整 `data-dir` 和实际使用的 `config.yml`。

不要只复制 `data.db`。SQLite 默认使用 WAL，运行期间还可能存在 `data.db-wal` 和 `data.db-shm`；最稳妥的做法是在停止进程后备份整个数据目录。

对于 MySQL，使用数据库自身的一致性备份工具，同时备份 `data-dir` 中的本地文件、脚本 Drive 和其他非数据库数据。

## Docker 升级

```bash
docker pull devld/go-drive
docker stop go-drive

# 备份示例；请把路径换成自己的数据目录
cp -a go-drive-data "go-drive-data.backup-$(date +%Y%m%d)"

docker rm go-drive
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  -v "$(pwd)/config.yml:/app/config.yml:ro" \
  --restart unless-stopped \
  devld/go-drive
```

使用 Compose 时：

```bash
docker compose pull
docker compose up -d
```

升级后检查登录、Drive 加载、权限、搜索和任务。数据库迁移在启动时执行；不要在升级失败后继续写入再直接覆盖旧数据库。

## 恢复

1. 停止 go-drive。
2. 保存当前失败现场，便于排查。
3. 恢复与该版本匹配的配置和完整数据目录/数据库备份。
4. 使用备份对应的 go-drive 版本启动。
5. 验证后再允许用户写入。

## 迁移到新服务器

- 保持 `data-dir` 的目录结构和文件权限。
- 如果启用了 `free-fs` 或挂载了宿主机绝对路径，同时迁移这些外部目录。
- 更新反向代理、DNS、OAuth 回调地址和 S3 CORS/防盗链域名。
- MySQL 用户需先迁移数据库，再迁移 `data-dir` 中的非数据库文件。
- 迁移后重新检查 WebDAV 客户端和脚本 Drive 的外部依赖。

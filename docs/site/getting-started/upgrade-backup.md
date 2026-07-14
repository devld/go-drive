---
title: Upgrade, Backup, and Restore
description: Upgrade go-drive safely and back up or restore its database, configuration, local files, sessions, scripts, and generated caches.
lang: en
translation_key: upgrade-backup
---

# Upgrade, Backup, and Restore

## Before upgrading

1. Read the target release notes.
2. Stop writes or schedule a maintenance window.
3. Stop go-drive so the database and WAL are fully flushed.
4. Back up the complete `data-dir` and the active `config.yml`.

Do not copy only `data.db`. SQLite uses WAL by default and `data.db-wal` and `data.db-shm` may exist while the process is running. The safest approach is to stop go-drive and back up the complete data directory.

For MySQL, use the database server's consistency-aware backup tools and separately back up local files, script drives, and other non-database content in `data-dir`.

## Upgrade with Docker

```bash
docker pull devld/go-drive
docker stop go-drive

# Example backup; replace the path with your real data directory
cp -a go-drive-data "go-drive-data.backup-$(date +%Y%m%d)"

docker rm go-drive
docker run -d --name go-drive \
  -p 8089:8089 \
  -v "$(pwd)/go-drive-data:/app/data" \
  -v "$(pwd)/config.yml:/app/config.yml:ro" \
  --restart unless-stopped \
  devld/go-drive
```

With Compose:

```bash
docker compose pull
docker compose up -d
```

After upgrading, verify sign-in, drive loading, permissions, search, and jobs. Database migrations run on startup. If an upgrade fails, do not continue writing data and then overwrite the database with an older copy.

## Restore

1. Stop go-drive.
2. Preserve the failed state for diagnosis.
3. Restore the configuration and complete data-directory/database backup that belong together.
4. Start the go-drive version that matches the backup.
5. Verify the instance before allowing writes.

## Migrate to another server

- Preserve the `data-dir` structure and file permissions.
- If `free-fs` is enabled or host absolute paths are mounted, migrate those external directories too.
- Update reverse proxy, DNS, OAuth callback, and S3 CORS/hotlink-protection domains.
- MySQL deployments should migrate the database and then the non-database files in `data-dir`.
- Recheck WebDAV clients and external dependencies used by script drives after migration.

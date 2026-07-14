---
title: FTP Drive
lang: en
translation_key: drive-ftp
---

# FTP Drive

| Field | Description | Default |
| --- | --- | --- |
| Host | FTP server hostname or IP address, without a port | Required |
| Port | FTP port | `21` |
| Username | Uses the anonymous user when empty | `anonymous` |
| Password | Uses the anonymous password when empty | `anonymous` |
| Concurrency | Maximum connection-pool concurrency | `5` |
| Timeout | Connection/operation timeout in Go duration format | `5s` |
| Cache TTL | Directory-entry cache time; zero or below disables caching | Disabled |

FTP does not encrypt transfers; prefer SFTP over public networks. Uploads, downloads, and generic copies pass through go-drive. Moving or renaming within the same FTP Drive uses remote rename.

Lower the concurrency setting if the server limits its number of connections. If directory listings become stale, clear the Drive cache or shorten the cache TTL.

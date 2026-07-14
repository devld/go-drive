---
title: SFTP Drive
description: Connect go-drive to an SFTP server using a password or SSH key, with host-key verification, root paths, and connection settings.
lang: en
translation_key: drive-sftp
---

# SFTP Drive

| Field | Description | Default |
| --- | --- | --- |
| Host | SSH/SFTP hostname or IP address | Required |
| Port | SSH port | `22` |
| Username | SSH user | Required |
| Password | Password authentication; use either this or a private key | Empty |
| Private key | PEM/OpenSSH private-key contents | Empty |
| Host public key | Pinned host key in SSH authorized-key format | Empty |
| Root path | Remote absolute path to map | `/` |
| Cache TTL | Directory-entry cache time; zero or below disables caching | Disabled |

In production, set **Host public key** to prevent man-in-the-middle attacks. You can use `ssh-keyscan` on a trusted network to obtain a candidate value, but verify its fingerprint through another trusted channel before saving it.

The root path must begin with `/`. File contents pass through go-drive. Move and rename use remote operations, while copy is handled by the generic go-drive job, which reads and uploads the file again.

---
title: MEGA Drive
description: Connect a MEGA cloud drive to go-drive with an email, password, and optional root folder. Two-factor codes are entered once during configuration and are not stored.
lang: en
translation_key: drive-mega
---

# MEGA Drive

| Field | Description | Default |
| --- | --- | --- |
| Email | MEGA account email | Required |
| Password | MEGA account password | Required |
| Root | Folder inside the cloud drive; empty uses the cloud drive root | Cloud drive root |
| Permanent delete | Permanently delete files instead of moving them to the MEGA rubbish bin | Off |
| HTTPS transfers | Use HTTPS for already encrypted file transfers | Off |

go-drive mounts the account's cloud drive. Inbox, rubbish, and incoming shares are not shown. File and folder names are decrypted locally. Uploads, downloads, and copies pass through go-drive because MEGA encrypts file contents on the client. Moving or renaming within this Drive uses MEGA's move and attribute commands.

MEGA allows more than one node with the same name in a folder. go-drive keeps the node with the latest modification time and hides the others. Names containing `/` or `\` are also hidden because they cannot be represented as a single path segment.

Two-factor authentication is required only when the MEGA account has it enabled. After the drive is saved, configuration signs in with the email and password. If MEGA asks for a code, enter it in that step. The code is used once and is not stored. A successful login stores the MEGA session in the Drive's private data. Later reloads reuse that session until the email or password changes, or MEGA rejects the session. Enter a fresh code only in that case.

**Permanent delete** is off by default, so deleted and replaced files move to the MEGA rubbish bin. They leave the cloud drive and can be restored from MEGA's own apps. Check it when removed files must not remain in the account. This Drive does not show the rubbish bin.

The API connection uses HTTPS. File bytes are encrypted before transfer, so MEGA recommends sending them over HTTP. Enable **HTTPS transfers** when the network blocks plain HTTP. There is no browser direct upload or download for this Drive.

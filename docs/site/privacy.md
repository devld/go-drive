---
title: Privacy and Data Processing
lang: en
translation_key: privacy
---

# Privacy and Data Processing

Last updated: July 14, 2026

go-drive is self-hosted open-source software, not a cloud storage service centrally operated by the project authors. In general, the person or organization deploying and managing a go-drive instance controls the data in that instance. The project authors cannot access accounts, files, or logs in independently operated instances.

This page explains the data the software itself may process. Instance operators should establish their own privacy policy based on their deployment region, users, enabled features, and third-party services.

## Data processed locally

go-drive may store the following in databases and data directories controlled by the operator:

- Usernames, password hashes, user groups, and root paths.
- Login sessions and their expiration times.
- Drive configurations, OAuth tokens, remote-service credentials, and script configurations.
- Permissions, mounts, path attributes, file buckets, and job configurations.
- File metadata, search indexes, thumbnails, and job execution logs.
- Files in local Drives and temporary files required for operations.

The exact locations depend on `data-dir`, `temp-dir`, and the database configuration. When MySQL or externally mounted directories are used, data may also be stored in those external systems.

## Browser storage

The web interface uses browser storage for the login token, list display preferences, navigation state, and information such as path passwords entered during the current session. Signing out revokes the current server-side session and clears the related login state. The browser may retain non-sensitive interface preferences.

## Logs

The server records runtime errors and diagnostic information, which may include usernames, client IP addresses, file paths, remote-service errors, and job script output. Enabling `GO_DRIVE_DEBUG` adds more diagnostic details. Instance operators should set suitable access controls, retention periods, and redaction rules.

## Third-party services

Data is sent to third parties only when the operator configures or a user invokes the corresponding feature:

- Storage backends such as OneDrive, Google Drive, S3, FTP, SFTP, and WebDAV receive authentication information, paths, file contents, and operation requests.
- LDAP/LDAPS servers receive login usernames and passwords for authentication and may return group membership.
- Microsoft and Google external file previewers receive a signed URL that can access the file.
- Administrator-installed script Drives, job scripts, uploaders, and shell thumbnail commands may access the network or process file contents.
- Reverse proxies, CDNs, databases, logging systems, and backup services process data according to the operator's configuration.

go-drive currently contains no advertising or general behavioral analytics service operated by the project authors.

## Data security and retention

The instance operator is responsible for:

- Using HTTPS, secure passwords, least privilege, and a trusted proxy configuration.
- Protecting database passwords, OAuth secrets, LDAP credentials, and file-bucket tokens stored in configuration.
- Deciding how long to retain databases, logs, indexes, thumbnails, temporary files, and backups.
- Promptly removing departed users, expired sessions, unused credentials, and data that is no longer needed.
- Testing backup and recovery and complying with applicable laws and regulations.

Deleting a user does not necessarily delete files they uploaded, job logs, backups, or data in remote cloud storage. Operators should perform complete data cleanup according to the applicable business relationship.

## User rights and contact

Users who want to access, correct, export, or delete data in an instance should contact the operator of that go-drive instance. The project authors generally cannot handle these requests on the operator's behalf because they cannot access self-hosted instances.

Software issues and security vulnerabilities can be reported through the project's [GitHub repository](https://github.com/devld/go-drive). Do not include personal data, passwords, tokens, or private file links in public issues.

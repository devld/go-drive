---
title: Security Guide
description: Secure a go-drive deployment by changing default credentials, limiting exposure, protecting secrets, and configuring trusted proxies.
lang: en
translation_key: security
---

# Security Guide

## Deployment checklist

- Change the default `admin` password.
- Use HTTPS. Never sign in or use WebDAV over plaintext HTTP on the public Internet.
- Expose only the reverse proxy through the firewall, not the go-drive listen port.
- Configure `trusted-proxies` precisely.
- Keep `free-fs: false` unless administrators are explicitly allowed to access every file visible to the process.
- Apply least privilege to anonymous users, normal users, group root paths, and path permissions.
- Keep anonymous WebDAV access disabled by default.
- Use LDAPS/StartTLS and certificate validation for LDAP.
- Back up the database and data directory regularly and test restores.
- Never expose database passwords, OAuth secrets, LDAP credentials, or file-bucket tokens in repositories, logs, or issues.

## Authentication and sessions

`auth.validity` controls session lifetime; `auto-refresh` extends active sessions. Signing out revokes the current session. Failed authentication callbacks are limited by client IP: after five failures in five minutes, requests receive HTTP 429. The reverse proxy must therefore report the real client IP correctly.

LDAP users can be provisioned just in time and have groups synchronized. An existing local account with the same username cannot be taken over by LDAP. Avoid reusing usernames across authentication sources.

## Signed file access

File-content and thumbnail URLs are signed. Their default lifetime is controlled by `signature-ttl: 12h`. A shorter value reduces the useful lifetime of a leaked link, but a value that is too short can interrupt long playback or downloads.

File-bucket downloads are a separate public-access mechanism and do not use normal user permission checks. Limit them with allowed types, maximum size, an unguessable upload token, referrer rules, and an appropriate cache policy.

## Path isolation

- A user's own root path overrides group root paths.
- Without a personal root path, the shallowest non-empty group root path is selected.
- Members of `admin` are not restricted by root paths.
- A root path defines the visible boundary; path permissions still control read/write access.
- Permissions on mounted content are resolved at the mount location.

`free-fs: true` lets administrators create local drives pointing to any absolute path accessible to the process. In containers, mount only the host directories that are genuinely needed.

## Custom code

Global styles, injected scripts, JavaScript drives, job scripts, and shell thumbnail handlers all execute code supplied by a trusted administrator. Only trusted administrators should modify them:

- Injected scripts run in every user's browser.
- JavaScript drives and jobs can make network requests and access mapped files.
- Shell thumbnail handlers run external commands with the go-drive process permissions.

Do not install script drives from unknown sources. Review both the drive script and its upload adapter.

## External viewers and OAuth

Microsoft/Google external viewers receive a signed file URL. Do not enable external viewers for internal or sensitive files.

OneDrive and Google Drive require an OAuth client secret. You may use the default static callback page or point `oauth-redirect-uri` at a compatible page you host. The URI configured in the OAuth console must match exactly.

## Logs and debugging

Error logs may contain paths, remote-service errors, and usernames. Treat centrally collected logs as sensitive data. Use `GO_DRIVE_DEBUG` only temporarily and disable it after troubleshooting.

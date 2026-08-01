---
title: Office Editing with WOPI
description: Connect go-drive to a self-hosted Collabora Online service for browser-based Office document viewing and editing.
lang: en
translation_key: wopi
---

# Office Editing with WOPI

go-drive implements the WOPI host endpoints needed to view and edit documents with a discovery-compatible Office service. The initial supported deployment target is self-hosted Collabora Online.

## Enable WOPI

Configure the discovery document exposed by the Office service:

```yaml
wopi:
  discovery-url: https://office.example.com/hosting/discovery
```

Restart go-drive. The Web UI obtains supported file extensions and `view`/`edit` actions from discovery and displays **Open in Office** for matching files. Missing or empty `discovery-url` disables WOPI.

The discovery endpoint must be reachable from the go-drive process. Conversely, the Office service must be able to reach every public go-drive domain used to open documents.

## Reverse proxy and multiple domains

go-drive creates a WOPISrc from the browser's current `Origin`, so one instance can be opened through more than one domain without a fixed public URL setting. For every public domain:

- Use HTTPS in production.
- Preserve the original `Host` header when proxying to go-drive.
- Route `api-path/wopi/*` to the same go-drive instance.
- Allow that host or alias in the Collabora `alias_groups` configuration.

The browser `Origin` must match the request `Host`. A proxy that rewrites `Host` to an internal service name will cause session creation to fail.

For a subpath deployment, configure `api-path` normally. For example, `api-path: /drive` produces WOPI endpoints below `/drive/wopi/`.

## Authentication and permissions

Only signed-in users can open the Office handler. A separate random WOPI token is issued for one user and one file; the normal go-drive login token is not sent to the Office service. Each WOPI callback reloads the user and passes through the normal user/group root, path-permission, and path-metadata wrappers.

WOPI sessions expire after 10 hours. Tokens and locks are process-local, so restarting go-drive invalidates open editors and requires users to reopen the document.

## Lock scope and external changes

WOPI locks coordinate WOPI clients only. They do not block:

- WebDAV writes;
- normal Web UI uploads or text editing;
- automated jobs; or
- direct changes in an underlying third-party storage service.

When a WOPI lock is created, go-drive records a version derived from the underlying entry's path, modification time, and size. If those values change before a WOPI save, `PutFile` returns a conflict instead of silently overwriting the external change. This is best-effort: a backend that doesn't report reliable modification times cannot provide complete conflict detection.

Locks expire after 30 minutes unless refreshed. Sessions and locks are not shared between multiple go-drive processes; use one application instance when WOPI editing is enabled.

## Compatibility boundary

The implementation includes CheckFileInfo, GetFile, PutFile, PutRelativeFile, Lock, GetLock, RefreshLock, Unlock, and UnlockAndRelock. It doesn't currently validate Microsoft proof keys. Microsoft 365 for the web also requires Cloud Storage Partner Program onboarding, registered domains, and stricter global conflict handling; use Collabora Online as the supported deployment target for this version.

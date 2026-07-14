---
title: Troubleshooting
description: Diagnose common go-drive startup, OAuth, storage, upload, WebDAV, reverse proxy, search, permission, cache, and thumbnail problems.
lang: en
translation_key: troubleshooting
---

# Troubleshooting

## A saved Drive does not take effect

After saving, click **Admin → Drives → Reload Drives**. If loading fails, inspect the backend logs. When editing a password or secret, keep the hidden placeholder supplied by the interface unless you intend to replace the credential.

## A local Drive reports that its path does not exist

- `free-fs: false`: enter a relative path. The current version creates `<data-dir>/local/<path>` automatically; check write permission on the data directory.
- `free-fs: true`: enter an absolute path that already exists inside the process or container; check that the host directory is mounted into the container.

## All users have the same IP behind a reverse proxy

Make sure the proxy sends `X-Forwarded-For` and add only the direct proxy IP/CIDR to `trusted-proxies`. Do not trust the entire internet. Restart after changing the configuration.

## Login returns 429

Five failures from the same client IP within five minutes trigger a temporary limit. Correct the password or LDAP configuration and wait for the window to end. If every user is limited together, `trusted-proxies` is usually misconfigured and the application sees only the proxy IP.

## Large-file upload fails

- Check Nginx `client_max_body_size`, timeouts, and `proxy_request_buffering`.
- Check free space and permissions for `temp-dir`.
- For direct S3/OneDrive uploads, check the browser console, CORS, and hotlink protection.
- For proxied transfers, check server outbound bandwidth and cloud-service timeouts.
- If empty files fail, make sure the proxy does not incorrectly remove or rewrite the request body.

## S3 upload fails

- Backend requests fail: check the endpoint, region, path-style mode, credentials, and bucket permissions.
- Only browser uploads fail: check CORS, the site origin, and the Referer allowlist.
- An internal network cannot access presigned URLs: enable proxy upload/download.

## OAuth fails

- The console redirect URI must exactly match `oauth-redirect-uri`.
- Check the Client ID, the Client Secret **value**, and its expiration.
- The OneDrive tenant, region, and account type must match.
- Google applications in testing status may have a restricted refresh-token lifetime.
- Server and browser clocks should be accurate.

## File listings are stale

After an external system changes files, clear the corresponding Drive cache or shorten/disable `cache_ttl`. Re-index the relevant paths to update search results as well.

## Search returns no results

1. Confirm `search.enabled: true` and `type: sqlite`, then restart.
2. Create an indexing job and wait for it to finish.
3. Check the `+`/`-` filtering rules.
4. Check the current user's root path and read permission.
5. Re-index after external changes.

## WebDAV connection fails

- The URL must contain the complete configured prefix and a trailing `/`.
- Use a go-drive username and password, not a browser token.
- Check the HTTPS certificate.
- The proxy must allow WebDAV methods.
- For a subpath deployment, `api-path`, the WebDAV prefix, and the proxy location must agree.
- Verify with curl or rclone before investigating operating-system client limitations.

## Thumbnails are not generated

- Check that the extension is listed in a handler's `file-types`.
- Check that the path-mapping tag exists on the corresponding handler.
- For a shell handler, check the command, `mime-type`, timeout, and whether the program is installed.
- For remote files using `write-content`, confirm read permission and network connectivity.
- Failures are cached; after fixing the cause, restart to clear the failure marker and retry.

## A job does not run

- Cron must use the standard five fields, without Quartz `?` or a seconds field.
- Check that the job is enabled, the process timezone, and the next run time.
- File events support only updated/deleted; check the path pattern.
- Inspect execution history and errors; verify the action with a manual run first.
- Prevent job output from matching its own event trigger.

## Reporting an issue

Include `go-drive -v` output, deployment method, operating system/architecture, database and Drive types, reproduction steps, and logs. Redact passwords, tokens, secrets, signed URLs, and personal paths from configuration.

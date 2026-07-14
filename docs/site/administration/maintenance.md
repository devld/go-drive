---
title: Maintenance and Runtime Status
lang: en
translation_key: maintenance
---

# Maintenance and Runtime Status

## Reload Drives

After adding, editing, deleting, or reauthorizing a Drive, click **Reload Drives** at the top of **Admin → Drives**. Saving only updates the database configuration; reloading replaces the running Drive instances.

If reloading fails, check the error log and the relevant remote credentials. Do not repeatedly save secret placeholder values: password fields use a hidden placeholder to protect existing secrets.

## Clear a Drive cache

Select a Drive under **Admin → Other → Clear Cache**. This is useful when:

- Files were changed directly in the remote system.
- OneDrive, Google Drive, S3, or another backend returns stale directory entries.
- A changed `cache_ttl` needs to take effect immediately.

Clearing a cache does not delete real files.

## Clean invalid permissions and mounts

After a Drive is deleted or a directory is moved externally, the database may retain permissions and mounts that point to paths that no longer exist. **Clean invalid permission/mount entries** deletes these records. Back up the database first because this is a persistent change.

## Search-index maintenance

The search page can create indexing jobs by path, show progress, and abort a job. Re-index affected paths after the Drive structure changes or files are modified externally. See [Search and Indexing](../features/search.html).

## System status

**Admin → Status** displays version and runtime statistics so you can confirm the version actually running and its resource usage. A troubleshooting report should include at least:

- The go-drive version and build revision.
- Docker or binary deployment and the operating-system architecture.
- Database type.
- The Drive type involved.
- Relevant configuration with credentials redacted.
- Backend logs from the same time period.

Set `GO_DRIVE_DEBUG=1` to temporarily add diagnostic information. Disable debug mode after the issue is resolved.

## Routine maintenance recommendations

- Regularly test backup restoration.
- Rotate OAuth client secrets before they expire.
- Review LDAP and S3 service-account permissions.
- Monitor data-directory, temporary-directory, and database capacity.
- Remove unused script Drives, job execution records, and thumbnail caches.
- Read the release notes and keep a rollback-capable backup before upgrading.

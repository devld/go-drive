---
title: Configuration Reference
description: Reference go-drive options for networking, databases, storage, search, WebDAV, thumbnails, automated jobs, and security.
lang: en
translation_key: configuration
---

# Configuration Reference

go-drive uses YAML configuration. It reads `config.yml` from the working directory by default, or a file selected with `-c`. Run `go-drive -show-config` to print the configuration after defaults have been applied.

The following example covers all current public settings. Keep unused features disabled or empty.

```yaml
listen: :8089

logging:
  level: info                 # debug, info, warn, or error

# Only proxies that connect directly to go-drive belong here
# trusted-proxies:
#   - 127.0.0.1
#   - 172.16.0.0/12

db:
  type: sqlite               # sqlite or mysql
  name: data.db              # SQLite filename or MySQL database name
  # host: 127.0.0.1
  # port: 3306
  # user: go_drive
  # password: change-me
  # config:
  #   loc: Local

data-dir: ./data
temp-dir: ""                 # Empty means data-dir/temp

drives-dir: script-drives
drive-uploaders-dir: drive-uploaders
drive-repository-url: https://api.github.com/repos/devld/go-drive/contents/script-drives

oauth-redirect-uri: https://go-drive.top/oauth_callback
max-concurrent-task: 100
free-fs: false
signature-ttl: 12h

thumbnail:
  ttl: 720h
  # concurrent: 4            # Defaults to max(CPU/2, 1)
  handlers:
    - type: image
      file-types: jpg,jpeg,png,gif,webp
    - type: text
      file-types: txt,md,xml,html,css,scss,js,json,jsx,properties,yml,yaml,ini,c,h,cpp,go,java,kt,gradle,ps1

# Archive preview supports zip, 7z, and rar files.
archive:
  # concurrent: 4            # Archive preview and zip packing, each defaults to 4
  max-size: 2g
  max-member-size: 512m
  max-entries: 100000
  # Sparse remote source cache limits used while inspecting an archive.
  cache-items: 128
  cache-size: 4g
  index-ttl: 24h
  # Complete decompressed member artifacts use a separate byte/TTL policy.
  content-cache-size: 4g
  content-cache-ttl: 24h
  # How long a generated zip download stays available.
  pack-ttl: 1m

auth:
  validity: 2h
  auto-refresh: true
  # providers: []

# web-dav:
#   enabled: true
#   prefix: /dav
#   allow-anonymous: false
#   max-cache-items: 1000

search:
  enabled: false
  type: sqlite

cache:
  type: mem
  clean-period: 10m

api-path: ""
web-path: ""
```

## Basic settings

| Setting | Default | Description |
| --- | --- | --- |
| `listen` | `:8089` | HTTP listen address |
| `logging.level` | `info` | Minimum log level; `GO_DRIVE_LOGGING_LEVEL` overrides it |
| `trusted-proxies` | Empty | Proxy IPs/CIDRs allowed to supply `X-Forwarded-For` |
| `data-dir` | `./data` | Database, local files, scripts, sessions, and cache data |
| `temp-dir` | `data-dir/temp` | Temporary files for upload, copy, and related work |
| `max-concurrent-task` | `100` | Concurrent copy, move, delete, and background tasks. The same number may wait; further submissions are rejected |
| `free-fs` | `false` | Allow local drives to use absolute paths; high risk |
| `signature-ttl` | `12h` | Lifetime of signed file-content and thumbnail URLs |
| `oauth-redirect-uri` | Project callback page | OAuth callback for OneDrive/Google Drive |
| `api-path` | Empty | Reverse-proxy subpath, for example `/drive` |
| `web-path` | Empty | Static asset path override; normally left empty |

The old `web-dir`, `lang-dir`, and `default-lang` settings have been removed. Release binaries embed the Web UI and language resources.

## Database

SQLite is suitable for a single application instance. The database is stored at `data-dir/<db.name>` and uses WAL with a five-second busy timeout by default.

MySQL requires at least `type`, `host`, `name`, `user`, and `password`. Values under `db.config` are passed to GORM as DSN parameters. Never commit real passwords to the repository.

Drive passwords, private keys, and OAuth tokens are sealed before they are written to the database. On first startup go-drive creates `data-dir/encryption.key` (mode `0600`) unless `GO_DRIVE_ENCRYPTION_KEY` is set. The environment variable and the file must contain the same secret when both exist. Back up whichever one you use together with the database; without it, stored drive secrets cannot be decrypted and have to be entered again.

## LDAP

Local password authentication is always available. Add LDAP as follows:

```yaml
auth:
  validity: 8h
  auto-refresh: true
  providers:
    - type: ldap
      config:
        url: ldaps://ldap.example.com:636
        start-tls: "false"
        skip-tls-verify: false
        bind-dn: cn=readonly,dc=example,dc=com
        bind-password: change-me
        base-dn: ou=users,dc=example,dc=com
        user-filter: "(uid=%s)"
        username-attr: uid
        group-base-dn: ou=groups,dc=example,dc=com
        group-filter: "(memberUid=%s)"
        group-name-attr: cn
        group-mapping:
          admin: ldap-admins
          staff: ldap-users,ldap-staff
```

- `%s` expands to the LDAP-escaped username/UID and suits `posixGroup`.
- `%d` expands to the full user DN and suits `groupOfNames` or AD, for example `(member=%d)`.
- Each `group-mapping` key is a go-drive group; its value lists one or more upstream groups.
- LDAP users are created on their first successful sign-in. When group search is configured, membership is synchronized on every sign-in.
- Usernames are exact and case-sensitive. Unknown users are tried against providers in configuration order.
- Accounts marked as external first use their provider, then fall back to their local password if that provider fails. JIT-created LDAP users do not receive a guessable local password.
- Avoid `skip-tls-verify: true`. Use LDAPS or StartTLS with a trusted certificate in production.

## Thumbnail handlers

Handler types are `image`, `text`, and `shell`. Shell handlers accept `shell`, `mime-type`, `write-content`, `max-size`, `timeout`, and related settings; see [Preview and thumbnails](../features/preview-thumbnail.html). The official Docker configuration enables libvips and ffmpeg. Extract the configuration from the image to get those templates.

## Archive preview

The built-in archive handler supports ZIP, 7z, and RAR. `max-size` limits the
compressed source inspected by the server; `max-member-size` limits one
decompressed member; and `max-entries` bounds the indexed member count.
`cache-items` and `cache-size` limit the sparse remote source cache used while
inspecting archives. The complete member index uses `index-ttl`. Decompressed
member preview bodies are complete artifacts and use the independent
`content-cache-size` byte budget and `content-cache-ttl` validity period.
`pack-ttl` is how long a generated zip download stays available after it is
created. `concurrent` limits archive preview tasks and zip packing tasks
separately; each group defaults to 4.

Archive inspection can continue in the background when a request reaches its
short response wait. The Web UI polls the task and opens the completed
artifact; closing the view only stops polling.

## WebDAV, search, and cache

- WebDAV is disabled by default. `allow-anonymous` remains subject to path permissions; test anonymous access before public deployment.
- The current search engine is `sqlite`; the old `bleve` setting is invalid.
- `web-dav.max-cache-items` limits the WebDAV file-object cache.
- The global `cache` currently uses an in-memory implementation; `clean-period` controls periodic cleanup.

See also:

- [Reverse proxy](./reverse-proxy.html)
- [Security guide](./security.html)
- [Search and indexing](../features/search.html)
- [WebDAV](../features/webdav.html)

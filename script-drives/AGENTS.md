# go-drive JavaScript Drive Adapter Guide

This file guides AI agents and human developers who add or modify storage adapters under `script-drives/`. The goal is to make a remote storage service behave like a go-drive directory tree by adding JavaScript files only, without recompiling go-drive.

## 1. Decide whether a script Drive is appropriate

### Good candidates

Prefer a script Drive when the service meets most of these conditions:

- It provides a stable HTTP/HTTPS REST API.
- Files and directories can be represented by a path, type, size, and modification time.
- Listing, uploading, downloading, directory creation, and deletion are available over HTTP.
- Authentication uses an API key, bearer token, HMAC signature, or OAuth 2.0.
- It does not require a Node.js package, native vendor SDK, dynamic library, or operating-system command.
- Each API operation can finish synchronously, or a remote operation can be polled until completion.
- Large files can be streamed, uploaded in parts, or uploaded directly from the browser.

Typical candidates include file APIs such as Dropbox, object-storage APIs such as Qiniu, self-hosted HTTP file services, and cloud drives that have a complete REST API but no built-in go-drive implementation.

### Technically possible, but usually not worthwhile

- WebDAV is HTTP-based, but the runtime has no DOM or XML parser. Use the built-in WebDAV Drive unless responses are exceptionally small and stable.
- S3-compatible storage can be signed with `Hash` / `Hmac` and `Bytes` encodings, but the built-in S3 Drive handles regions, multipart uploads, and compatibility differences more reliably.
- A service exposed only through a vendor JavaScript SDK is viable only if the SDK can run in the Goja ES6 runtime without Node.js or DOM dependencies. Calling the REST API directly is usually better.
- A synchronous long-poll endpoint can work, but occupies a VM while polling. Check `ctx.err()` and apply a timeout.

### Poor candidates

Implement these as Go Drives, or use an existing built-in Drive:

- Samba/SMB/CIFS, SFTP, and FTP require raw TCP, session negotiation, connection reuse, or binary protocols. The script runtime has no socket API. SMB also requires negotiation, signing, encryption, and stateful handles; `http()` cannot substitute for it.
- Local filesystems, FUSE, block devices, and tape systems require operating-system or device access.
- SDKs that require native libraries, external commands, Node.js `require`, `Buffer`, streams, or npm packages.
- Services that require WebSocket, HTTP/2-specific flow control, client certificates, or a custom transport stack without an equivalent ordinary HTTP API.
- Workloads requiring heavy CPU processing, complex compression/encryption, or large in-memory buffers. The Goja runtime is not designed for them.
- Services that cannot reliably list a hierarchy, read file contents, or expose stable paths.

Rule of thumb: use a script Drive when the core task is “construct HTTP requests and map JSON to `EntryRecord` objects.” Use Go when the core task is “implement a transport protocol, integrate with the operating system, or reuse a native SDK.”

## 2. Sources of truth to read before editing

Check these sources in order. Do not rely only on old adapter examples:

1. `docs/scripts/env/drive.d.ts` — Drive lifecycle, interfaces, and Drive-specific APIs.
2. `docs/scripts/global.d.ts` — global HTTP, IO, error, encoding, path, and form APIs.
3. `script-drives/jsconfig.json` and `script-drives/jsconfig.uploader.json` — load the server-adapter and browser-uploader declarations respectively.
4. `drive/script/helper.js` — required methods, method binding, and the actual behavior of `$` shared properties.
5. `drive/script/index.go` and `drive/script/utils.go` — Go/JavaScript value conversion, entry cache, write-path eviction, and resource ownership.
6. `script-drives/dropbox.js` — OAuth, pagination, streaming uploads, and temporary download URLs.
7. `script-drives/qiniu.js` and `qiniu-uploader.js` — HMAC signing, object storage, and direct browser uploads.
8. `docs/drive-uploaders/types.d.ts` — required when implementing direct browser uploads.

Type declarations assist development; the Go bridge is the final source of truth. If the declarations and implementation disagree, correct the declaration or documentation instead of inventing an API.

## 3. Deliverables and file conventions

The server-side adapter is:

```text
script-drives/<name>.js
```

The optional browser-side direct uploader is:

```text
script-drives/<name>-uploader.js
```

Use a stable, short, lowercase identifier for `<name>`. Files with the same base name form one extension. A server script must begin with consecutive `//` metadata lines:

```js
// @name Example Cloud
// @version 1.0.0
// @description Example Cloud REST API adapter.
//
// Create an API token with file read/write permissions.
```

- `// @name Example Cloud` is the display name shown in the UI.
- `// @version 1.0.0` is required. The main server script version is the version of the extension; bump it when the uploader changes.
- Scripts without `@name` or `@version` are ignored when listing installed scripts or syncing the repository.
- An optional `// @uploader example-uploader.js` line names the browser uploader in the same repository.
- `// @description` starts the Markdown description. Following `//` lines that are not `@` directives continue the description until a blank non-comment line.
- Editor types for server scripts come from `script-drives/jsconfig.json`; browser uploaders use `script-drives/jsconfig.uploader.json` (no `/// <reference>` needed).
- After saving a script, create or reload the Drive from the administration UI.

## 4. Runtime constraints

### JavaScript version

Server scripts run in Goja with ES6+ syntax (`let`/`const`, arrow functions, classes, rest/spread, and modern array methods). The runtime does not provide:

- `import`, `export`, or `require`;
- DOM APIs, `window`, `fetch`, or `XMLHttpRequest`;
- Node.js `Buffer`, `process`, `fs`, `crypto`, or npm packages.

Goja provides standard `Promise` objects, but all go-drive host APIs are synchronous and do not return Promises. There is no JavaScript event loop, timer API, or asynchronous HTTP API; top-level `await` is not supported. ES6 standard objects, JSON, Date, RegExp, and the go-drive APIs declared in the `.d.ts` files are available. `dayjs` is built in.

Built-in go-drive globals and utility objects are immutable. After a pooled VM finishes initialization, the global object is frozen. Do not monkey-patch or extend `http`, `console`, `pathUtils`, `urlUtils`, `dayjs`, or other host-provided bindings. Objects and arrays that originate in Go (`urlUtils.parse`, `resp.json()`, `cache.getEntry`, `createInstance` `config`, `$` reads, `$event`) are read-only; copy with object/array spread before changing them.

Browser uploader scripts have a separate runtime and may use modern JavaScript, Promise, Blob, FormData, and browser APIs. Never mix browser APIs into the server script.

### Synchronous calls, concurrency, and state

- `http()` is synchronous. A method occupies one VM until it returns.
- go-drive maintains a VM pool and may call one Drive concurrently. Do not assume call order or use ordinary mutable globals as shared state.
- The object returned by `createInstance` is frozen after methods are bound. Ordinary properties assigned there should be treated as read-only configuration.
- Only instance properties whose names begin with `$` are synchronized between VMs through go-drive shared storage.
- A `$` value must be JSON-serializable. Objects and arrays are read as copies. Mutating a nested value does not persist it; reassign the complete `$` property.
- A single shared-property read or write is protected, but a read-modify-write sequence is not atomic. Prefer concurrency control provided by the remote API.
- Never place response bodies, readers, or functions in a `$` property.
- Periodic work uses `intervals` / `onInterval`. Go holds the clock; ticks borrow a VM like `get`. Do not occupy a VM with `sleep` loops.

```js
const next = this.$state;
next.count += 1;
this.$state = next; // Reassignment writes the complete value back.
```

Administrators may configure the VM pool as `MaxTotal,MaxIdle,MinIdle,IdleTime`; its default is `100,50,10,30m`. An adapter must not depend on a particular pool size.

### Cancellation and resources

- Host I/O (`http`, OAuth, Drive) uses the VM run context. Cancelling the Go task interrupts the VM; there is no JavaScript `Context` object.
- `HttpResponse.text()` reads the complete body and disposes the response.
- If `text()` is not called, call `dispose()`. The usual exception is returning a successful `resp.body` directly from `getReader` or `getThumbnail`; go-drive then owns and closes it.
- Explicitly obtained `ReadCloser` and `TempFile` values must be closed after use.
- Never call `readAsString()` for a large upload. Pass the Reader to `http()` or upload it in parts.

### Paths and EntryRecord objects

- The root path is always the empty string `""`. Other paths never start with `/`.
- Return normalized `/`-separated paths. Use `pathUtils.join/parent/base/clean`, not operating-system path rules.
- `get("")` is served by the runtime as a directory `EntryRecord`. Do not special-case the root in `get`.
- `list(path)` returns direct children only. It neither includes the listed directory nor recurses.
- File `size` is in bytes; use `-1` when unknown. Directory size is normally `-1`.
- `modTime` is Unix time in milliseconds; use `-1` when unknown. Do not return seconds.
- Omitting `meta` defaults to `{readable: true, writable: true}`. A read-only Drive or entry must explicitly set `writable: false`.
- Store only small string values needed by native copy/move in `data` (remote file ids, revisions). Never store tokens or signed URLs there. Instance ownership is detected by the runtime; do not put a drive id in `data`.

A normal `EntryRecord` looks like:

```js
{
  isDir: false,
  path: "folder/file.txt",
  size: 123,
  modTime: 1710000000000,
  meta: { readable: true, writable: true },
  data: { id: "remote-id" }
}
```

### Errors

Use these Error subclasses for expected failures:

- `new BadRequestError(message)` — invalid user input or configuration.
- `new NotFoundError(message)` — a missing path. `get` must map a remote 404 to this error.
- `new NotAllowedError(message)` — insufficient permissions, conflict, or prohibited operation.
- `new UnsupportedError(message)` — an unavailable capability; selected callers may apply a fallback.
- `new RemoteApiError(status, message)` — other remote API failures.

Use `instanceof` when catching errors (`e instanceof NotFoundError`). Never include tokens, secrets, Authorization headers, complete signed URLs, or private response bodies in errors or logs.

## 5. Lifecycle

Define the adapter with `defineDrive(setup, methods)`.

`setup` is evaluated before any Drive instance exists (`configForm`, `initConfig`, `init`, `validateConfig`, `createInstance`). `methods` run on the created instance; `this` is inferred from the object returned by `createInstance` (`DriveThis<T>`). `$` properties on that object are typed as cross-VM shared JSON state.

### `configForm` / `initConfig` / `init`

`configForm` is the static admin form. It is always an array, and its field names must not begin with `_`; those names are reserved by the Script Drive wrapper. Required fields are saved as part of the Drive config before initialization.

`initConfig(config, utils)` is optional and is called after the static config has been saved. It returns the same `DriveInitConfiguration` shape as a native Drive, including a dynamic `form`, its current `value`, `configured`, and optional `oauth`. Use `utils.data.load("key", ...)` to inspect only the previously saved dynamic fields needed for the current step and return different forms for later steps. Dynamic form field names must also not begin with `_`.

`init(data, config, utils)` is optional and receives the submitted dynamic data. It is responsible for saving dynamic values with `utils.data.save`, or for calling the low-level OAuth helpers. Empty strings are passed through unchanged; saving an empty string clears that key from the data store.

OAuth is explicit: call `utils.oauthInitConfig`, `utils.oauthInit`, and `utils.oauthLoad` from `initConfig` / `init` / `createInstance`. There is no automatic OAuth request/principal hook. See `dropbox.js`.

`validateConfig(config)` runs before `createInstance` and validates the static config.

Include `entryCacheTTLFormItem("2h")` when users should set the entry cache TTL. Pass the raw form value through as `entryCacheTTL` from `createInstance`; the runtime accepts duration strings or `ms(...)`. Omit / `""` / `undefined` / `null` / `<= 0` disables caching. The form item is not inserted automatically.

### `createInstance(config, utils)` (required)

Return instance state from the static config, loading only the dynamic fields needed by the Drive through `utils.data.load("key", ...)`: credentials, clients, `entryCacheTTL: config.cache_ttl`, and optional `writable: false` for a read-only Drive (`writable` defaults to `true`). Optional `intervals` declare Drive-local periodic work (see `onInterval`). The runtime attaches `this.cache` and Drive methods, then freezes the object. `$` properties remain shared across VMs. Entry cache lookup, write-path eviction, root `get("")`, copy/move ownership, and default `meta` / `upload` / `getReader` run in Go so cache hits do not occupy a VM.

Required methods: `get` and `list`, plus `getReader` or `getURL`. `upload` defaults to `useLocalProvider`. `getReader` defaults to `new UnsupportedError()` when `getURL` exists. `meta` defaults to `{ writable: this.writable !== false }`.

## 6. Drive method contracts

### Required methods

#### `meta() -> DriveMeta`

Optional. Defaults to `{ writable: instance.writable !== false }`.

#### `get(path) -> EntryRecord`

Return the `EntryRecord` at one non-root path. The Go runtime serves `get("")` and caches successful results (including `data`) using `entryCacheTTL` without entering the JS VM on hit. A missing path must throw `new NotFoundError()`.

#### `list(path) -> EntryRecord[]`

Return all direct children. Handle every remote page, marker, or cursor rather than returning only the first page. Return `[]` for an empty directory.

#### `getReader(entry, start, size) -> ReadCloser`

Read file content. `start === -1 && size === -1` means the complete content. For range reads, send an appropriate Range header and validate the response status. If `getURL` is implemented, omit `getReader`; the runtime throws `new UnsupportedError()`.

### Write methods

#### `save(path, size, override, reader, progress)`

Stream the Reader to the remote service. Go reports `size` as the task total before the call and passes a loaded-only `ProgressReporter`. Progress is explicit: pass `reader.withProgress(progress)` to `http()` or `HttpFormData`. Do not report the same Reader at more than one layer.

Honor `override`. Prefer a conditional remote write over a check-then-write sequence that introduces a race. Do not evict caches or return `get`; the runtime evicts the target and parent, then re-gets.

#### `makeDir(path)`

Create one directory. The dispatcher ensures that parents exist. Object storage may create a zero-byte object with a trailing `/`; if the service has implicit directories, follow its native semantics.

#### `delete(path, progress) -> void`

Delete the path and all descendants. If remote directory deletion is not recursive, enumerate with `buildEntriesTree` and `flattenEntriesTree`, using a total-only derived reporter while planning, then call `progress.addLoaded(delta)` after successful batches.

### Native copy and move

#### `copy(from, to, override, progress)`

The runtime calls this only when `from` belongs to this Drive instance. `from` is an `EntryRecord` (`path`, `isDir`, `size`, `modTime`, `data`). Throw `new UnsupportedError()` when native copy is unavailable (for example directories). The dispatcher will fall back to reading the source and calling destination `save`. Never disguise an actual remote failure as Unsupported.

#### `move(from, to, override, progress)`

Same ownership wrapping as `copy`. `new UnsupportedError()` from `move` does **not** trigger automatic copy-and-delete.

### Upload strategy

#### `upload(path, size, override, config) -> DriveUploadConfig | undefined`

Chooses the frontend upload strategy; it does not replace `save`. Defaults to `useLocalProvider(size)`.

- Return `useCustomProvider(safeConfig)` for direct browser uploads (no uploader name).
- After a successful browser upload the runtime calls this again with `config.action === "Completed"` and evicts the target and parent. Return immediately for that action unless the Drive must finish a server-side commit.
- `config` sent to the browser is fully visible to the user. Include only short-lived, least-privilege upload credentials, never a long-lived secret.

### Downloads and thumbnails

#### `getURL(entry) -> ContentURL` (optional)

Return:

```js
{
  url: "https://...",
  header: { Authorization: "Bearer ..." }, // Optional
  proxy: true,                             // Optional
  downloadFileName: "name.txt"            // Optional
}
```

With no `header` and `proxy: false`, the client receives a redirect. If a `header` is present, proxying is forced, or `proxy: true`, go-drive proxies the response. Private headers are not exposed to the browser. Do not cache a short-lived signed URL in `entry.data`.

#### `getThumbnail(entry) -> ReadCloser | ContentURL` (optional)

Return a remote thumbnail response body or URL configuration. When returning the body, do not dispose it first. Mark eligible entries with `meta.selfThumbnail: true` in `get`/`list` (type, extension, size only; no network). Omit `getThumbnail` when the service has no thumbnail capability.

### Background intervals

#### `onInterval(name)` (optional)

Required when `createInstance` returns `intervals`. Go owns the clock; the callback runs on a borrowed VM and must finish within `timeout` (default `30s`). Return `"25m"` or `ms(...)` to choose the next delay; omit to keep `interval`.

```js
createInstance: function (config, utils) {
  return {
    oauth: utils.oauthLoad(oauthReq(utils.config), {
      clientID: config.client_id,
      clientSecret: config.client_secret
    }),
    intervals: [{ name: "refresh", interval: "30m", immediately: false }]
  };
}

onInterval: function (name) {
  if (name !== "refresh") return;
  this.oauth.refresh();
}
```

Do not call `oauthLoad` inside `onInterval`. Standard OAuth request paths should keep using `token()`.

## 7. Available JavaScript APIs

The following runtime surface is safe to depend on. Refer to the two `.d.ts` files for exact field types.

### Configuration, state, and cache

- `utils.config`: `oauthRedirectURI`, `version`, `revHash`, and `buildAt`.
- `utils.data.load(...keys)` / `utils.data.save(map)`: persistent string configuration.
- `this.cache`: entry cache created for the instance. Use it only for extra invalidation; `get`/`list` and write methods are wrapped automatically.
- `parseDuration(value)`: `ms(...)` or a duration string (`"2s"`, `"2d3h"`). Empty → `0`; invalid throws TypeError.
- `DriveCache.putEntry`, `putEntries`, and `putChildren` (`ttl` is `ms(...)` or a duration string).
- `DriveCache.getEntry` and `getChildren`; a miss returns `null`.
- `DriveCache.evict(path, descendants)` and `evictAll()`.
- Cross-VM shared state: assign `$` properties on the instance (`this.$foo = …`).
- `selfDrive`: a host `Drive` wrapping the current script Drive (`selfDrive instanceof Drive`). Use `get`/`save`/`makeDir`/`copy`/`move`/`list`/`delete`. Distinct from `this` (`DriveThis`).

### OAuth

- `utils.oauthInitConfig(request, credentials)`: produce a configuration/OAuth step and possibly an existing `OAuthHolder`. The result is read-only; copy fields into a new object if `initConfig` needs to change `configured` / `oauth.principal`. `OAuthHolder.token` / `refresh` on `oauthHolder` still work.
- `utils.oauthInit(data, request, credentials)`: handle the OAuth callback during initialization.
- `utils.oauthLoad(request, credentials)`: construct the runtime `OAuthHolder` from a stored token.
- `OAuthHolder.token()`: retrieve an automatically refreshed token.
- `OAuthHolder.refresh()`: force a token-endpoint exchange even if the access token is still valid. Call this from `onInterval` on the holder created in `createInstance`. Do not call `oauthLoad` again.
- An OAuth request contains `endpoint`, `redirectUrl`, `scopes`, and `text`; credentials contain `clientID` and `clientSecret`.
- Endpoint authentication styles are `OAuthStyle.AutoDetect`, `InParams`, and `InHeader`. Prefer auto-detection unless the provider requires otherwise.

Follow `dropbox.js`. Do not persist OAuth state manually or duplicate refresh-token logic. Ordinary request methods should keep using `token`; `refresh` is for keep-alive when a provider expires unused refresh tokens.

### HTTP

- `http(url, { method, headers, body, timeout }?) -> HttpResponse`; `method` defaults to GET. Allowed methods are HEAD, GET, POST, PUT, DELETE, PATCH, and OPTIONS. `timeout` is `ms(...)` or a duration string and defaults to `"30s"`; `0` disables it (the VM run context still applies). Other host network APIs do not add a timeout. Uploads that may run longer than 30s must set `timeout: 0` (or a longer duration).
- The body may be a Reader, string, Bytes, or HttpFormData.
- For a Reader body, `Content-Length` comes from `headers` when set (the body is truncated to that size). Otherwise a known size is used (`TempFile` remaining bytes, including `limitReader` wrapping one) so object-storage PUT is not chunked. String and Bytes always use their actual length. HttpFormData is multipart. Set `Transfer-Encoding: chunked` to skip auto `Content-Length`. Wrap a Reader with `reader.withProgress(progress)` when its consumption should update the Go task.
- `new HttpFormData()`, with `appendField` and `appendFile`.
- `HttpResponse.status`, `body`, `bodySize()`, `text()`, `json()`, and `dispose()`.
- `HttpResponse.headers.get(key)`, `values(key)`, and `getAll()`.

The HTTP client does not follow redirects automatically. Handle 3xx responses according to the service API. For every unexpected status, read or dispose the response and map it to a go-drive error.

### Logging and debugging

- `console.debug/error/info/log/warn(...)`: write to the server log.

Use the appropriate `console.debug/info/warn/error` level directly, and redact
arguments before constructing the log message.

### IO

- `new Bytes(size)` allocates zeroed bytes. `Bytes.fromString(s)` copies UTF-8. Bytes has `length`, `slice(start, end)` with an exclusive end, and `toString(encoding?, options?)` (`utf8`, `hex`, `base64`, `base64url`; `options.padded` defaults to `true`).
- `Bytes.fromHex(s)`, `Bytes.fromBase64(s, options?)`, `Bytes.fromBase64Url(s, options?)`, and `Bytes.random(n)` (CSPRNG; `n` in `[0, 1MiB]`).
- Reader has `read(bytes)` (returns `-1` at EOF), `readAsString()`, and `limitReader(n)`.
- ReadCloser additionally has `close()`.
- `new TempFile()`; TempFile has all Reader methods plus `write(bytes)`, `copyFrom(reader)`, `seekTo(offset, whence)`, `size()`, and `close()`. `copyFrom` only reports progress when its Reader argument was explicitly wrapped with `withProgress`.
- Host values are JS classes: `value instanceof Bytes`, `tmp instanceof TempFile && tmp instanceof Reader`, `selfDrive instanceof Drive`, `entry instanceof Entry`. `defineDrive` `get`/`list` return plain `EntryRecord` objects, not host `Entry`.
- `SEEK_START`, `SEEK_CURRENT`, and `SEEK_END`.

### Progress and synchronization

- Write methods receive an operation-scoped `ProgressReporter` as the last argument. `addLoaded(delta)` and `addTotal(delta)` are incremental. `derive({loaded, total})` may only remove permissions. `save` receives a loaded-only reporter because Go establishes its total; Reader consumption reports only when explicitly wrapped with `withProgress`.
- `sleep(duration)`.
- Drive intervals: `createInstance` may return `intervals: [{ name, interval, timeout?, immediately? }]`. Go schedules them; `onInterval(name)` runs on a borrowed VM. `interval` / `timeout` / the return value are `ms(...)` or duration strings (`"30m"`); timeout defaults to `"30s"`. Overlapping ticks are skipped. A returned duration reschedules the next run; omitting it keeps `interval`. Stopped when the Drive is disposed. Do not emulate this with `sleep` loops or Admin Jobs.
- `ms(milliseconds)` converts milliseconds to a Go Duration.

### Paths, time, encoding, and hashes

- `pathUtils.clean/join/parent/base/ext/isRoot`.
- `urlUtils.parse(url)` and `urlUtils.build(parts)` use Go's `net/url`; `urlUtils.parseSearchParams` and `urlUtils.buildSearchParams` handle query strings; `searchParams` maps keys to string arrays. Parsed objects are read-only; copy with `Object.assign` / spread before changing fields and calling `build`.
- `dayjs`; Go `time.Time` values appear as JavaScript `Date`.
- `new Hash("md5")` / `"sha1"` / `"sha256"` / `"sha512"`; Hash has `write`, `writeFrom`, and `sum`.
- `new Hmac("sha1", keyBytes)` returns a streaming Hmac (`hmac instanceof Hash`).
- `writeFrom` hashes from the current offset to EOF (it does not seek to the start). On a seekable `TempFile` it restores that same offset, so hashing from the middle (`seekTo` then `writeFrom`) and then continuing from that point works. After `write`, call `seekTo(0, SEEK_START)` if the whole file must be hashed or uploaded. One-shot Readers (response bodies) are consumed; copy them to a TempFile first if they must be reused. When a digest must appear in request headers (`Content-MD5`), hash the TempFile, then `http()`.

### Traversal helpers

- `buildEntriesTree(entry, byteProgress?, progress?)`.
- `flattenEntriesTree(node, deepFirst?)`.
- `findEntries(rootDrive, pattern, bytesProgress?, progress?)`.
- `Entry` (host class from `Drive.list` / `selfDrive.get`, not `EntryRecord`): getters `path/name/type/size/meta/modTime/unwrap/data/drive`; methods `getUrl/getReader`.

### Forms

Supported types are `md`, `textarea`, `text`, `password`, `checkbox`, `checkboxes`, `select`, `path`, `form`, and `code`. Drive credentials normally need only text/password/select/checkbox. Use `type: "password"` for secrets. When an existing secret is returned through `DriveInitConfiguration.value`, the admin API replaces it with a reserved placeholder; submitting that unchanged placeholder preserves the stored secret.

Common fields are `label/type/field/required/description/disabled/defaultValue`. A select uses `options`, a path uses `pathOptions`, a nested form uses `forms`, and a code editor uses `code`. Use the lowerCamel Go-bridge field names declared in the `.d.ts` files.

## 8. Minimal complete example

This example assumes the remote service provides:

- `GET /v1/entries?path=...` returning `{entry: RemoteEntry}`;
- `GET /v1/children?path=...&cursor=...` returning `{items: [], nextCursor: ""}`;
- `PUT /v1/content?path=...&override=true|false` accepting a file stream;
- `POST /v1/directories` accepting `{"path":"..."}`;
- `DELETE /v1/entries?path=...&recursive=true`;
- `GET /v1/download-url?path=...` returning a short-lived `{url: "..."}`.

It demonstrates the interface contract and does not represent a real service:

```js
// @name Example REST Drive
// @version 1.0.0
// @uploader example-uploader.js
// @description Example of a complete HTTP API based adapter.
//
// Enter the API endpoint and a token with file read/write permissions.

defineDrive(
  {
    configForm: [
      { label: "API URL", field: "base_url", type: "text", required: true },
      { label: "Token", field: "token", type: "password", required: true },
      entryCacheTTLFormItem("5m")
    ],

    validateConfig(config) {
      if (!/^https:\/\/[^/]+(?:\/.*)?$/.test(config.base_url || "")) {
        throw new BadRequestError("API URL must use HTTPS");
      }
    },

    createInstance(config) {
      return {
        entryCacheTTL: config.cache_ttl,
        baseURL: config.base_url.replace(/\/+$/, ""),
        token: config.token
      };
    }
  },
  {
    get(path) {
      const result = requestJson(
        this,
        "GET",
        "/v1/entries?path=" + encodeURIComponent(path)
      );
      return toEntry(result.entry);
    },

    list(path) {
      const all = [];
      let cursor = "";
      do {
        let route = "/v1/children?path=" + encodeURIComponent(path);
        if (cursor) route += "&cursor=" + encodeURIComponent(cursor);
        const page = requestJson(this, "GET", route);
        all.push(...page.items.map(toEntry));
        cursor = page.nextCursor || "";
      } while (cursor);
      return all;
    },

    save(path, size, override, reader, progress) {
      const route = "/v1/content?path=" + encodeURIComponent(path) +
        "&override=" + (override ? "true" : "false");
      const resp = http(this.baseURL + route, {
        method: "PUT",
        headers: {
          Authorization: "Bearer " + this.token,
          "Content-Type": "application/octet-stream"
        },
        body: reader.withProgress(progress),
        timeout: 0
      });
      const status = resp.status;
      const message = resp.text();
      if (status === 409) throw new NotAllowedError("destination already exists");
      if (status < 200 || status >= 300) throw new RemoteApiError(status, message);
    },

    makeDir(path) {
      requestJson(this, "POST", "/v1/directories", { path });
    },

    copy(from, to, override, progress) {
      throw new UnsupportedError();
    },

    move(from, to, override, progress) {
      throw new UnsupportedError();
    },

    delete(path, progress) {
      requestJson(
        this,
        "DELETE",
        "/v1/entries?recursive=true&path=" + encodeURIComponent(path)
      );
    },

    getURL(entry) {
      const data = requestJson(
        this,
        "GET",
        "/v1/download-url?path=" + encodeURIComponent(entry.path)
      );
      return { url: data.url };
    }
  }
);


function requestJson(drive, method, route, body) {
  const headers = {
    Authorization: "Bearer " + drive.token,
    Accept: "application/json"
  };
  let payload;
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  const resp = http(drive.baseURL + route, {
    method,
    headers,
    body: payload
  });
  const status = resp.status;
  const text = resp.text();
  let data = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (e) {
      throw new RemoteApiError(status, "remote returned invalid JSON");
    }
  }
  if (status === 404) throw new NotFoundError();
  if (status === 401 || status === 403) throw new NotAllowedError("remote denied access");
  if (status < 200 || status >= 300) {
    throw new RemoteApiError(status, data.message || "remote request failed");
  }
  return data;
}

function toEntry(remote) {
  return {
    isDir: remote.type === "dir",
    path: pathUtils.clean(remote.path),
    size: remote.type === "dir" ? -1 : remote.size,
    modTime: remote.modified_at ? dayjs(remote.modified_at).valueOf() : -1,
    data: { id: String(remote.id) }
  };
}
```

A real adapter must add service-specific pagination, upload behavior, redacted errors, and native copy/move where available. Do not blindly replace the example URLs.

## 9. Direct browser uploader

Add an uploader only when all of these are true: the remote service supports browser CORS; the server can issue short-lived least-privilege credentials; and relaying large files through go-drive is a real bottleneck.

The server-side `upload` method returns:

```js
return useCustomProvider({
  uploadURL: signed.url,
  token: signed.shortLivedToken
});
```

`example-uploader.js` must call `defineUploader`:

```js
defineUploader({
  chunkSize: 5 * 1024 * 1024,
  async start(ctx) {
    if (ctx.chunks === 1) return null;
    const res = await ctx.request({ method: "post", url: ctx.config.initURL });
    return { uploadId: res.data.uploadId };
  },
  async upload(ctx, args) {
    return ctx.request({
      method: "put",
      url: ctx.config.uploadURL,
      data: args.blob,
      onUploadProgress: args.onProgress
    });
  },
  async complete(ctx, args) { /* commit multipart if args.session is set */ },
  async abort(ctx, args) { /* delete remote upload if args.session is set */ }
});
```

The runtime slices the file, calls `abort` on failure/cancel (not after success), and notifies the Drive with `{ action: "Completed" }` after `complete`. Use `ctx.request` so pause/cancel abort in-flight uploads. Follow `qiniu-uploader.js`. Verify CORS preflight, failed requests, cancellation cleanup, empty files, a non-full final chunk, and expired credentials. Long-lived access keys or secret keys must never enter browser configuration.

## 10. Implementation and acceptance workflow

An agent must proceed in this order:

1. Read the target service's official API. Record authentication, metadata, pagination, upload, download, directory, copy, move, delete, rate-limit, and error semantics.
2. Perform the suitability assessment first. If the service is unsuitable, explain why it needs a Go Drive instead of generating a plausible-looking placeholder script.
3. Define one unambiguous remote-object-to-`EntryRecord` mapping, including root path, directory emulation, and time units.
4. Implement the configuration lifecycle and least-privilege credentials.
5. Implement `get`/`list` and downloads (`getURL` or `getReader`), then write methods.
6. Implement native copy/move only when the remote service truly supports them.
7. Review pagination, status handling, response disposal, cache invalidation, and cancellation paths.
8. Implement and review a browser uploader separately when required.
9. Run:

   ```sh
   go test ./script ./drive/script
   ```

10. On a test instance, verify root and empty directories, pagination, Unicode and space-containing paths, 404, empty/small/large files, overwrite and no-overwrite, deep directory creation, recursive delete, same-Drive and cross-Drive copy, move, cancellation, timeout, 401/403, 429, 5xx, credential refresh, and cache consistency.
11. Set `GO_DRIVE_LOGGING_LEVEL=debug` only for temporary diagnosis. Confirm logs are redacted, then remove it.
12. Do not edit `build/`, `web/dist/`, or dependency directories, and never commit real credentials.

Completion means every contract above is implemented or has explicit Unsupported behavior, and read/write operations, errors, cancellation, and resource ownership have been verified against a real test instance. Merely loading the script is not sufficient.

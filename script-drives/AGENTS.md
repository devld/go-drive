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
- Each API operation can finish synchronously, or an asynchronous operation can be polled until completion.
- Large files can be streamed, uploaded in parts, or uploaded directly from the browser.

Typical candidates include file APIs such as Dropbox, object-storage APIs such as Qiniu, self-hosted HTTP file services, and cloud drives that have a complete REST API but no built-in go-drive implementation.

### Technically possible, but usually not worthwhile

- WebDAV is HTTP-based, but the runtime has no DOM or XML parser. Use the built-in WebDAV Drive unless responses are exceptionally small and stable.
- S3-compatible storage can be signed with `encUtils`, but the built-in S3 Drive handles regions, multipart uploads, and compatibility differences more reliably.
- A service exposed only through a vendor JavaScript SDK is viable only if the SDK can be rewritten as ES5 without Node.js or DOM dependencies. Calling the REST API directly is usually better.
- Long-polling asynchronous APIs can work, but occupy a VM while polling. Check `ctx.Err()` and apply a timeout.

### Poor candidates

Implement these as Go Drives, or use an existing built-in Drive:

- Samba/SMB/CIFS, SFTP, and FTP require raw TCP, session negotiation, connection reuse, or binary protocols. The script runtime has no socket API. SMB also requires negotiation, signing, encryption, and stateful handles; `http()` cannot substitute for it.
- Local filesystems, FUSE, block devices, and tape systems require operating-system or device access.
- SDKs that require native libraries, external commands, Node.js `require`, `Buffer`, streams, or npm packages.
- Services that require WebSocket, HTTP/2-specific flow control, client certificates, or a custom transport stack without an equivalent ordinary HTTP API.
- Workloads requiring heavy CPU processing, complex compression/encryption, or large in-memory buffers. The ES5 interpreter is not designed for them.
- Services that cannot reliably list a hierarchy, read file contents, or expose stable paths.

Rule of thumb: use a script Drive when the core task is “construct HTTP requests and map JSON to Entry objects.” Use Go when the core task is “implement a transport protocol, integrate with the operating system, or reuse a native SDK.”

## 2. Sources of truth to read before editing

Check these sources in order. Do not rely only on old adapter examples:

1. `docs/scripts/env/drive.d.ts` — Drive lifecycle, interfaces, and Drive-specific APIs.
2. `docs/scripts/global.d.ts` — global HTTP, IO, error, encoding, path, and form APIs.
3. `drive/script/helper.js` — required methods, method binding, and the actual behavior of `$` shared properties.
4. `drive/script/index.go` and `drive/script/utils.go` — Go/JavaScript value conversion and resource ownership.
5. `script-drives/dropbox.js` — OAuth, pagination, streaming uploads, and temporary download URLs.
6. `script-drives/qiniu.js` and `qiniu-uploader.js` — HMAC signing, object storage, and direct browser uploads.
7. `docs/drive-uploaders/types.d.ts` — required when implementing direct browser uploads.

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
// Example Cloud
// Example Cloud REST API adapter.
//
// Create an API token with file read/write permissions.

/// <reference path="../docs/scripts/env/drive.d.ts"/>
```

- The first line is the display name shown in the UI.
- Following `//` lines, up to the empty comment line, are the Markdown description.
- The `reference` directive provides editor completion only; it does not change runtime behavior.
- After saving a script, create or reload the Drive from the administration UI.

## 4. Runtime constraints

### JavaScript version

Server scripts run in Otto and must be ES5. Do not use:

- `let`, `const`, arrow functions, classes, template literals, destructuring, or spread syntax;
- `async`/`await`, Promise, generators;
- `import`, `export`, or `require`;
- DOM APIs, `window`, `fetch`, or `XMLHttpRequest`;
- Node.js `Buffer`, `process`, `fs`, `crypto`, or npm packages.

ES5 standard objects, JSON, Date, RegExp, and the go-drive APIs declared in the `.d.ts` files are available. `dayjs` is built in.

Browser uploader scripts have a separate runtime and may use modern JavaScript, Promise, Blob, FormData, and browser APIs. Never mix browser APIs into the server script.

### Synchronous calls, concurrency, and state

- `http()` is synchronous. A method occupies one VM until it returns.
- go-drive maintains a VM pool and may call one Drive concurrently. Do not assume call order or use ordinary mutable globals as shared state.
- The object returned by `defineCreate` is frozen. Ordinary properties assigned by its constructor should be treated as read-only configuration.
- Only instance properties whose names begin with `$` are synchronized between VMs through go-drive shared storage.
- A `$` value must be JSON-serializable. Objects and arrays are read as copies. Mutating a nested value does not persist it; reassign the complete `$` property.
- A single shared-property read or write is protected, but a read-modify-write sequence is not atomic. `newLocker()` protects only its current VM and is not a cross-VM lock. Prefer concurrency control provided by the remote API.
- Never place response bodies, readers, contexts, or functions in a `$` property.

```js
var next = this.$state;
next.count += 1;
this.$state = next; // Reassignment writes the complete value back.
```

Administrators may configure the VM pool as `MaxTotal,MaxIdle,MinIdle,IdleTime`; its default is `100,50,10,30m`. An adapter must not depend on a particular pool size.

### Contexts and resources

- Use the method's `ctx` for every remote request. Do not use `newContext()` for normal requests.
- Call `ctx.Err()` inside pagination, polling, and multipart loops so cancellation is noticed promptly.
- A context returned by `newContextWithTimeout(parent, timeout)` must call `Cancel()` on every path.
- `HttpResponse.Text()` reads the complete body and disposes the response.
- If `Text()` is not called, call `Dispose()`. The usual exception is returning a successful `resp.Body` directly from `getReader` or `getThumbnail`; go-drive then owns and closes it.
- Explicitly obtained `ReadCloser` and `TempFile` values must be closed after use.
- Never call `ReadAsString()` for a large upload. Pass the Reader to `http()` or upload it in parts.

### Paths and Entry objects

- The root path is always the empty string `""`. Other paths never start with `/`.
- Return normalized `/`-separated paths. Use `pathUtils.join/parent/base/clean`, not operating-system path rules.
- `get("")` must return the root-directory Entry.
- `list(path)` returns direct children only. It neither includes the listed directory nor recurses.
- File `Size` is in bytes; use `-1` when unknown. Directory size is normally `-1`.
- `ModTime` is Unix time in milliseconds; use `-1` when unknown. Do not return seconds.
- Omitting `Meta` defaults to `{Readable: true, Writable: true}`. A read-only Drive or Entry must explicitly set `Writable: false`.
- Store only small string values needed by native copy/move in `Data`. At minimum, mark instance ownership with a persistent instance ID. Never store tokens or signed URLs there.

A normal Entry looks like:

```js
{
  IsDir: false,
  Path: "folder/file.txt",
  Size: 123,
  ModTime: 1710000000000,
  Meta: { Readable: true, Writable: true },
  Data: { d: this._instanceID, id: "remote-id" }
}
```

### Errors

Use these constructors for expected failures:

- `ErrBadRequest(message)` — invalid user input or configuration.
- `ErrNotFound(message)` — a missing path. `get` must map a remote 404 to this error.
- `ErrNotAllowed(message)` — insufficient permissions, conflict, or prohibited operation.
- `ErrUnsupported(message)` — an unavailable capability; selected callers may apply a fallback.
- `ErrRemoteApi(status, message)` — other remote API failures.

Use the matching `isBadRequestErr`, `isNotFoundErr`, `isNotAllowedErr`, `isUnsupportedErr`, and `isRemoteApiErr` predicates when catching errors. Never include tokens, secrets, Authorization headers, complete signed URLs, or private response bodies in errors or logs.

## 5. Lifecycle

### `defineInitConfig(fn)` — build configuration steps (optional)

Signature:

```js
function (ctx, config, utils) -> DriveInitConfiguration
```

- Load saved values with `utils.Data.Load(...)`.
- Return `Configured`, `Form`, and optional `Value` and `OAuth` fields.
- This stage may be called repeatedly and must be idempotent.
- Do not create long-lived connections or depend on in-memory side effects here.

Without this function, go-drive displays only the script description and the initial state remains unconfigured. Define it explicitly for most adapters.

### `defineInit(fn)` — validate and persist configuration (optional)

Signature:

```js
function (ctx, data, config, utils) -> void
```

Validate submitted data, perform an OAuth code exchange or other initialization, and persist values with `utils.Data.Save(data)`. `Save` merges keys; it does not remove omitted fields automatically.

Generate and persist an `_id` for each Drive instance. This ID determines whether an Entry passed to copy/move belongs to the same remote account.

### `defineCreate(fn)` — create the runtime instance (required)

Signature:

```js
function (ctx, config, utils) -> Drive
```

Load credentials from `utils.Data`, verify that configuration is complete, and construct read-only fields, a cache, and any required `$` shared state. The returned object must implement `meta`, `get`, `list`, and `getReader`; otherwise Drive creation fails.

## 6. Drive method contracts

### Required methods

#### `meta(ctx) -> DriveMeta`

Return capabilities for the whole Drive. A writable Drive returns `{Writable: true}`; a read-only Drive returns `{Writable: false}`. Do not rely on implicit conversion of an empty return value.

#### `get(ctx, path) -> Entry`

Return the Entry at one path. Construct the root locally. A missing non-root path must throw `ErrNotFound()`. A cache lookup may happen first; store the Entry after a successful remote request.

#### `list(ctx, path) -> Entry[]`

Return all direct children. Handle every remote page, marker, or cursor rather than returning only the first page. Call `ctx.Err()` in the loop. Return `[]` for an empty directory.

#### `getReader(ctx, entry, start, size) -> ReadCloser`

Read file content. `start === -1 && size === -1` means the complete content. For range reads, send an appropriate Range header and validate the response status. If a working `getURL` is implemented, this method may always throw `ErrUnsupported()` because downloads and generic copies prefer the URL. The function itself is still required.

### Write methods

#### `save(ctx, path, size, override, reader) -> Entry`

Stream the Reader to the remote service and report total size and progress:

```js
ctx.Total(size, true);
var body = reader.ProgressReader(ctx);
```

Honor `override`. Prefer a conditional remote write over a check-then-write sequence that introduces a race. On success, evict the target and parent-directory caches, then call `get` and return the final Entry.

#### `makeDir(ctx, path) -> Entry`

Create one directory. The dispatcher ensures that parents exist. Object storage may create a zero-byte object with a trailing `/`; if the service has implicit directories, follow its native semantics. Evict the parent cache on success.

#### `delete(ctx, path) -> void`

Delete the path and all descendants. If remote directory deletion is not recursive, enumerate with `buildEntriesTree` and `flattenEntriesTree`, then delete depth-first. Evict the target including descendants and its parent on success.

### Native copy and move

#### `copy(ctx, from, to, override) -> Entry`

Perform native copy only when the source Entry belongs to this Drive instance and the remote API supports server-side copy:

```js
var source = from.Unwrap();
var data = source.Data();
if (!data || data.d !== this._instanceID) throw ErrUnsupported();
```

Throw `ErrUnsupported()` when native copy is unavailable. The dispatcher will fall back to reading the source and calling destination `save`, recursively for directories. Never disguise an actual remote failure as Unsupported.

#### `move(ctx, from, to, override) -> Entry`

Likewise, call `Unwrap()` and verify instance ownership first. On success, evict the source, target, and both parent-directory caches.

Important: `ErrUnsupported()` from `move` does **not** trigger automatic copy-and-delete. It reports that cross-Drive move is unsupported. Implement native move when the product needs it. Do not implement recursive copy-and-delete inside the adapter unless partial failures and data-loss risks are handled explicitly.

### Upload strategy

#### `upload(ctx, path, size, override, config) -> DriveUploadConfig | undefined`

This method chooses the frontend upload strategy; it does not replace `save`:

- Normally return `useLocalProvider(size)`. Small files stream through go-drive; large files are first uploaded to go-drive in chunks and then passed to `save`.
- Return `useCustomProvider("name", safeConfig)` for direct browser uploads.
- A browser uploader can call `uploadCallback(data)` to invoke this method again. Handle `config.action` to finish a multipart upload, evict caches, or return the result.
- `Config` sent to the browser is fully visible to the user. Include only short-lived, least-privilege upload credentials, never a long-lived secret.

### Downloads and thumbnails

#### `getURL(ctx, entry) -> ContentURL` (optional)

Return:

```js
{
  URL: "https://...",
  Header: { Authorization: "Bearer ..." }, // Optional
  Proxy: true,                             // Optional
  DownloadFileName: "name.txt"            // Optional
}
```

With no Header and `Proxy: false`, the client receives a redirect. If a Header is present, proxying is forced, or `Proxy: true`, go-drive proxies the response. Private headers are not exposed to the browser. Do not cache a short-lived signed URL in Entry.Data.

#### `hasThumbnail(entry) -> boolean` (optional)

This must be a quick, local predicate with no network request. Usually check the entry type, extension, and size.

#### `getThumbnail(ctx, entry) -> ReadCloser | ContentURL` (optional)

Return a remote thumbnail response body or URL configuration. When returning the body, do not dispose it first. Omit both thumbnail methods when the service has no thumbnail capability.

## 7. Available JavaScript APIs

The following runtime surface is safe to depend on. Refer to the two `.d.ts` files for exact field types.

### Configuration, state, and cache

- `utils.Config`: `OAuthRedirectURI`, `Version`, `RevHash`, and `BuildAt`.
- `utils.Data.Load(...keys)` / `utils.Data.Save(map)`: persistent string configuration.
- `utils.CreateCache()`: create the current Drive's Entry cache.
- `DriveCache.PutEntry`, `PutEntries`, and `PutChildren`.
- `DriveCache.GetEntry` and `GetChildren`; a miss returns `null`.
- `DriveCache.Evict(path, descendants)` and `EvictAll()`.
- `setData(map)` / `getData(key)`: low-level shared state, normally accessed through `$property`.
- `selfDrive`: the Go wrapper of the current script Drive, with Get/Save/MakeDir/Copy/Move/List/Delete methods.

### OAuth

- `utils.OAuthInitConfig(request, credentials)`: produce a configuration/OAuth step and possibly an existing authorization response.
- `utils.OAuthInit(ctx, data, request, credentials)`: handle the OAuth callback during initialization.
- `utils.OAuthGet(request, credentials)`: construct the runtime response wrapper.
- `OAuthResponse.Token()`: retrieve an automatically refreshed token.
- An OAuth request contains Endpoint, RedirectURL, Scopes, and Text; credentials contain ClientID and ClientSecret.
- Endpoint authentication styles are `OAuthStyle.AutoDetect`, `InParams`, and `InHeader`. Prefer auto-detection unless the provider requires otherwise.

Follow the complete sequence in `dropbox.js`. Do not persist OAuth state manually or duplicate refresh-token logic.

### HTTP

- `http(ctx, method, url, headers?, body?) -> HttpResponse`; methods are HEAD, GET, POST, PUT, DELETE, PATCH, and OPTIONS.
- The body may be a Reader, string, Bytes, or HttpFormData.
- `newFormData()`, with `AppendField` and `AppendFile`.
- `HttpResponse.Status`, `Body`, `BodySize()`, `Text()`, and `Dispose()`.
- `HttpResponse.Headers.Get(key)`, `Values(key)`, and `GetAll()`.

The HTTP client does not follow redirects automatically. Handle 3xx responses according to the service API. For every unexpected status, read or dispose the response and map it to a go-drive error.

### Logging and debugging

- `DEBUG`: whether `GO_DRIVE_DEBUG` is enabled.
- `console.debug/error/info/log/warn(...)`: write to the server log.
- `consoleWrite(level, ...messages)`: low-level logging; normally use `console`.

Log only inside a `DEBUG` branch, and redact arguments before constructing the log message.

### IO

- `newBytes(string)` and `newEmptyBytes(size)`; Bytes has `Len()`, `Slice(start, end)` with an exclusive end, and `String()`.
- Reader has `Read(bytes)` (returns `-1` at EOF), `ReadAsString()`, `LimitReader(n)`, and `ProgressReader(ctx)`.
- ReadCloser additionally has `Close()`.
- `newTempFile()`; TempFile has all Reader methods plus `Write(bytes)`, `CopyFrom(reader)`, `SeekTo(offset, whence)`, `Size()`, and `Close()`.
- `SEEK_START`, `SEEK_CURRENT`, and `SEEK_END`.

### Context, progress, and synchronization

- Context has `Err()`; a timeout context also has a required `Cancel()`.
- TaskCtx has `Progress(value, absolute)` and `Total(value, absolute)`.
- `newContext()`, `newContextWithTimeout(parent, ms(...))`, and `newTaskCtx(ctx, callback)`.
- `sleep(duration)`; `newLocker()` returns a current-VM mutex with `Lock()` and `Unlock()`.
- `ms(milliseconds)` converts milliseconds to a Go Duration.

### Paths, time, encoding, and hashes

- `pathUtils.clean/join/parent/base/ext/isRoot`.
- `dayjs` and `toDate(goTime)`; GoTime also has `UnixMilli()`.
- `encUtils.toHex/fromHex/base64Encode/base64Decode/urlBase64Encode/urlBase64Decode`.
- `encUtils.newHash(HASH.*)`; Hasher has `Write` and `Sum`.
- `encUtils.hmac(HASH.*, payloadBytes, keyBytes)`.
- HASH supports MD5, SHA1, SHA256, and SHA512.

### Traversal helpers

- `buildEntriesTree(ctx, entry, byteProgress?)`.
- `flattenEntriesTree(node, deepFirst?)`.
- `findEntries(ctx, rootDrive, pattern, bytesProgress?)`.
- DriveEntry methods: `Path/Name/Type/Size/Meta/ModTime/GetURL/GetReader/Unwrap/Data/Drive`.

### Forms

Supported types are `md`, `textarea`, `text`, `password`, `checkbox`, `checkboxes`, `select`, `path`, `form`, and `code`. Drive credentials normally need only text/password/select/checkbox. Use `Type: "password"` for secrets.

Common fields are `Label/Type/Field/Required/Description/Disabled/DefaultValue`. A select uses `Options`, a path uses `PathOptions`, a nested form uses `Forms`, and a code editor uses `Code`. Use the capitalized Go-bridge field names declared in the `.d.ts` files.

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
// Example REST Drive
// Example of a complete HTTP API based adapter.
//
// Enter the API endpoint and a token with file read/write permissions.

/// <reference path="../docs/scripts/env/drive.d.ts"/>

function form() {
  return [
    { Label: "API URL", Field: "base_url", Type: "text", Required: true },
    { Label: "Token", Field: "token", Type: "password", Required: true }
  ];
}

defineInitConfig(function (ctx, config, utils) {
  var data = utils.Data.Load("base_url", "token");
  return {
    Configured: !!(data.base_url && data.token),
    Form: form(),
    Value: data
  };
});

defineInit(function (ctx, data, config, utils) {
  if (!/^https:\/\/[^/]+(?:\/.*)?$/.test(data.base_url || "")) {
    throw ErrBadRequest("API URL must use HTTPS");
  }
  data.base_url = data.base_url.replace(/\/+$/, "");
  var saved = utils.Data.Load("_id");
  if (!saved._id) data._id = String(Math.round(Math.random() * 1000000000));
  utils.Data.Save(data);
});

defineCreate(function (ctx, config, utils) {
  var data = utils.Data.Load("base_url", "token", "_id");
  if (!data.base_url || !data.token || !data._id) {
    throw ErrNotAllowed("drive not configured");
  }
  return new ExampleDrive(data, utils.CreateCache());
});

function ExampleDrive(data, cache) {
  this._baseURL = data.base_url;
  this._token = data.token;
  this._instanceID = data._id;
  this._cache = cache;
  this._cacheTTL = ms(5 * 60 * 1000);
}

function requestJSON(drive, ctx, method, route, body) {
  var headers = {
    Authorization: "Bearer " + drive._token,
    Accept: "application/json"
  };
  var payload;
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  var resp = http(ctx, method, drive._baseURL + route, headers, payload);
  var status = resp.Status;
  var text = resp.Text();
  var data = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (e) {
      throw ErrRemoteApi(status, "remote returned invalid JSON");
    }
  }
  if (status === 404) throw ErrNotFound();
  if (status === 401 || status === 403) throw ErrNotAllowed("remote denied access");
  if (status < 200 || status >= 300) {
    throw ErrRemoteApi(status, data.message || "remote request failed");
  }
  return data;
}

function toEntry(drive, remote) {
  return {
    IsDir: remote.type === "dir",
    Path: pathUtils.clean(remote.path),
    Size: remote.type === "dir" ? -1 : remote.size,
    ModTime: remote.modified_at ? dayjs(remote.modified_at).valueOf() : -1,
    Data: { d: drive._instanceID, id: String(remote.id) }
  };
}

function cacheEntry(item) {
  return {
    IsDir: item.Type === "dir",
    Path: item.Path,
    Size: item.Size,
    ModTime: item.ModTime,
    Data: item.Data
  };
}

ExampleDrive.prototype.meta = function (ctx) {
  return { Writable: true };
};

ExampleDrive.prototype.get = function (ctx, path) {
  if (pathUtils.isRoot(path)) {
    return { IsDir: true, Path: "", Size: -1, ModTime: -1 };
  }
  var cached = this._cache.GetEntry(path);
  if (cached) return cacheEntry(cached);
  var result = requestJSON(
    this,
    ctx,
    "GET",
    "/v1/entries?path=" + encodeURIComponent(path)
  );
  var entry = toEntry(this, result.entry);
  this._cache.PutEntry(entry, this._cacheTTL);
  return entry;
};

ExampleDrive.prototype.list = function (ctx, path) {
  var cached = this._cache.GetChildren(path);
  if (cached) return cached.map(cacheEntry);
  var all = [];
  var cursor = "";
  do {
    ctx.Err();
    var route = "/v1/children?path=" + encodeURIComponent(path);
    if (cursor) route += "&cursor=" + encodeURIComponent(cursor);
    var page = requestJSON(this, ctx, "GET", route);
    for (var i = 0; i < page.items.length; i++) {
      all.push(toEntry(this, page.items[i]));
    }
    cursor = page.nextCursor || "";
  } while (cursor);
  this._cache.PutChildren(path, all, this._cacheTTL);
  return all;
};

ExampleDrive.prototype.save = function (ctx, path, size, override, reader) {
  ctx.Total(size, true);
  var route = "/v1/content?path=" + encodeURIComponent(path) +
    "&override=" + (override ? "true" : "false");
  var resp = http(
    ctx,
    "PUT",
    this._baseURL + route,
    {
      Authorization: "Bearer " + this._token,
      "Content-Type": "application/octet-stream"
    },
    reader.ProgressReader(ctx)
  );
  var status = resp.Status;
  var message = resp.Text();
  if (status === 409) throw ErrNotAllowed("destination already exists");
  if (status < 200 || status >= 300) throw ErrRemoteApi(status, message);
  this._cache.Evict(path, false);
  this._cache.Evict(pathUtils.parent(path), false);
  return this.get(ctx, path);
};

ExampleDrive.prototype.makeDir = function (ctx, path) {
  requestJSON(this, ctx, "POST", "/v1/directories", { path: path });
  this._cache.Evict(path, false);
  this._cache.Evict(pathUtils.parent(path), false);
  return this.get(ctx, path);
};

ExampleDrive.prototype.copy = function (ctx, from, to, override) {
  throw ErrUnsupported(); // The dispatcher falls back to a streamed copy.
};

ExampleDrive.prototype.move = function (ctx, from, to, override) {
  throw ErrUnsupported(); // Move has no automatic fallback.
};

ExampleDrive.prototype.delete = function (ctx, path) {
  requestJSON(
    this,
    ctx,
    "DELETE",
    "/v1/entries?recursive=true&path=" + encodeURIComponent(path)
  );
  this._cache.Evict(path, true);
  this._cache.Evict(pathUtils.parent(path), false);
};

ExampleDrive.prototype.upload = function (ctx, path, size, override, config) {
  return useLocalProvider(size);
};

ExampleDrive.prototype.getReader = function (ctx, entry, start, size) {
  throw ErrUnsupported(); // Content remains available through getURL.
};

ExampleDrive.prototype.getURL = function (ctx, entry) {
  var data = requestJSON(
    this,
    ctx,
    "GET",
    "/v1/download-url?path=" + encodeURIComponent(entry.Path)
  );
  return { URL: data.url };
};
```

A real adapter must add service-specific pagination, upload behavior, redacted errors, and native copy/move where available. Do not blindly replace the example URLs.

## 9. Direct browser uploader

Add an uploader only when all of these are true: the remote service supports browser CORS; the server can issue short-lived least-privilege credentials; and relaying large files through go-drive is a real bottleneck.

The server-side `upload` method returns:

```js
return useCustomProvider("example", {
  uploadURL: signed.url,
  token: signed.shortLivedToken
});
```

The complete `example-uploader.js` file must evaluate to a callable factory function. The factory receives an UploadFactoryContext and returns:

- required `upload(blob, seq, onProgress)`;
- optional `prepare() -> chunk count`;
- optional `getChunk(seq)`;
- optional `complete()`;
- optional `onCleanup()` to cancel a remote multipart upload;
- writable `ctx.maxConcurrent`;
- HTTP through `ctx.request(...)`, allowing go-drive to track and cancel it;
- optional `ctx.uploadCallback({action: "Completed"})` after completion so the server can confirm the upload and evict caches.

Follow `qiniu-uploader.js`. Verify CORS preflight, failed requests, cancellation cleanup, empty files, a non-full final chunk, and expired credentials. Long-lived access keys or secret keys must never enter browser configuration.

## 10. Implementation and acceptance workflow

An agent must proceed in this order:

1. Read the target service's official API. Record authentication, metadata, pagination, upload, download, directory, copy, move, delete, rate-limit, and error semantics.
2. Perform the suitability assessment first. If the service is unsuitable, explain why it needs a Go Drive instead of generating a plausible-looking placeholder script.
3. Define one unambiguous remote-object-to-Entry mapping, including root path, directory emulation, and time units.
4. Implement the configuration lifecycle and least-privilege credentials.
5. Implement `meta/get/list/getReader` and downloads, then write methods.
6. Implement native copy/move only when the remote service truly supports them.
7. Review pagination, status handling, response disposal, cache invalidation, and cancellation paths.
8. Implement and review a browser uploader separately when required.
9. Run:

   ```sh
   go test ./script ./drive/script
   ```

10. On a test instance, verify root and empty directories, pagination, Unicode and space-containing paths, 404, empty/small/large files, overwrite and no-overwrite, deep directory creation, recursive delete, same-Drive and cross-Drive copy, move, cancellation, timeout, 401/403, 429, 5xx, credential refresh, and cache consistency.
11. Enable `GO_DRIVE_DEBUG=1` only for temporary diagnosis. Confirm logs are redacted, then disable it.
12. Do not edit `build/`, `web/dist/`, or dependency directories, and never commit real credentials.

Completion means every contract above is implemented or has explicit Unsupported behavior, and read/write operations, errors, cancellation, and resource ownership have been verified against a real test instance. Merely loading the script is not sufficient.

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
4. `drive/script/index.go` and `drive/script/utils.go` — Go/JavaScript value conversion, entry cache, write-path eviction, and resource ownership.
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
// @name Example Cloud
// @version 1.0.0
// @description Example Cloud REST API adapter.
//
// Create an API token with file read/write permissions.

/// <reference path="../docs/scripts/env/drive.d.ts"/>
```

- `// @name Example Cloud` is the display name shown in the UI.
- `// @version 1.0.0` is required. The main server script version is the version of the extension; bump it when the uploader changes.
- Scripts without `@name` or `@version` are ignored when listing installed scripts or syncing the repository.
- An optional `// @uploader example-uploader.js` line names the browser uploader in the same repository.
- `// @description` starts the Markdown description. Following `//` lines that are not `@` directives continue the description until a blank non-comment line.
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
- The object returned by `createInstance` is frozen after methods are bound. Ordinary properties assigned there should be treated as read-only configuration.
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
- `get("")` is served by the runtime as a directory Entry. Do not special-case the root in `get`.
- `list(path)` returns direct children only. It neither includes the listed directory nor recurses.
- File `Size` is in bytes; use `-1` when unknown. Directory size is normally `-1`.
- `ModTime` is Unix time in milliseconds; use `-1` when unknown. Do not return seconds.
- Omitting `Meta` defaults to `{Readable: true, Writable: true}`. A read-only Drive or Entry must explicitly set `Writable: false`.
- Store only small string values needed by native copy/move in `Data` (remote file ids, revisions). Never store tokens or signed URLs there. Instance ownership is detected by the runtime; do not put a drive id in `Data`.

A normal Entry looks like:

```js
{
  IsDir: false,
  Path: "folder/file.txt",
  Size: 123,
  ModTime: 1710000000000,
  Meta: { Readable: true, Writable: true },
  Data: { id: "remote-id" }
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

Define the adapter with `defineDrive(setup, methods)`.

`setup` is evaluated before any Drive instance exists (`configForm`, `initConfig`, `init`, `validateConfig`, `createInstance`). `methods` run on the created instance; `this` is inferred from the object returned by `createInstance` (`DriveThis<T>`). `$` properties on that object are typed as cross-VM shared JSON state.

### `configForm` / `initConfig` / `init`

`configForm` is the static admin form. It is always an array, and its field names must not begin with `_`; those names are reserved by the Script Drive wrapper. Required fields are saved as part of the Drive config before initialization.

`initConfig(ctx, config, utils)` is optional and is called after the static config has been saved. It returns the same `DriveInitConfiguration` shape as a native Drive, including a dynamic `Form`, its current `Value`, `Configured`, and optional `OAuth`. Use `utils.Data.Load("key", ...)` to inspect only the previously saved dynamic fields needed for the current step and return different forms for later steps. Dynamic form field names must also not begin with `_`.

`init(ctx, data, config, utils)` is optional and receives the submitted dynamic data. It is responsible for saving dynamic values with `utils.Data.Save`, or for calling the low-level OAuth helpers. Empty strings are passed through unchanged; saving an empty string clears that key from the data store.

OAuth is explicit: call `utils.OAuthInitConfig`, `utils.OAuthInit`, and `utils.OAuthLoad` from `initConfig` / `init` / `createInstance`. There is no automatic OAuth request/principal hook. See `dropbox.js`.

`validateConfig(config)` runs before `createInstance` and validates the static config.

Include `entryCacheTTLFormItem("2h")` when users should set the entry cache TTL. Pass the raw form value through as `entryCacheTTL` from `createInstance`; the Go runtime parses `ms`/`s`/`m`/`h`. Empty, invalid, or `<= 0` disables caching. The form item is not inserted automatically.

### `createInstance(ctx, config, utils)` (required)

Return instance state from the static config, loading only the dynamic fields needed by the Drive through `utils.Data.Load("key", ...)`: credentials, clients, `entryCacheTTL: config.cache_ttl`, and optional `writable: false` for a read-only Drive (`writable` defaults to `true`). `ctx` is the Drive-creation context; use it for any remote requests during setup. The runtime attaches `this.cache` and Drive methods, then freezes the object. `$` properties remain shared across VMs. Entry cache lookup, write-path eviction, root `get("")`, copy/move ownership, and default `meta` / `upload` / `getReader` run in Go so cache hits do not occupy a VM.

Required methods: `get` and `list`, plus `getReader` or `getURL`. `upload` defaults to `useLocalProvider`. `getReader` defaults to `ErrUnsupported()` when `getURL` exists. `meta` defaults to `{ Writable: this.writable !== false }`.

## 6. Drive method contracts

### Required methods

#### `meta(ctx) -> DriveMeta`

Optional. Defaults to `{ Writable: instance.writable !== false }`.

#### `get(ctx, path) -> Entry`

Return the Entry at one non-root path. The Go runtime serves `get("")` and caches successful results (including `Data`) using `entryCacheTTL` without entering the JS VM on hit. A missing path must throw `ErrNotFound()`.

#### `list(ctx, path) -> Entry[]`

Return all direct children. Handle every remote page, marker, or cursor rather than returning only the first page. Call `ctx.Err()` in the loop. Return `[]` for an empty directory.

#### `getReader(ctx, entry, start, size) -> ReadCloser`

Read file content. `start === -1 && size === -1` means the complete content. For range reads, send an appropriate Range header and validate the response status. If `getURL` is implemented, omit `getReader`; the runtime throws `ErrUnsupported()`.

### Write methods

#### `save(ctx, path, size, override, reader)`

Stream the Reader to the remote service and report total size and progress:

```js
ctx.Total(size, true);
var body = reader.ProgressReader(ctx);
```

Honor `override`. Prefer a conditional remote write over a check-then-write sequence that introduces a race. Do not evict caches or return `get`; the runtime evicts the target and parent, then re-gets.

#### `makeDir(ctx, path)`

Create one directory. The dispatcher ensures that parents exist. Object storage may create a zero-byte object with a trailing `/`; if the service has implicit directories, follow its native semantics.

#### `delete(ctx, path) -> void`

Delete the path and all descendants. If remote directory deletion is not recursive, enumerate with `buildEntriesTree` and `flattenEntriesTree`, then delete depth-first.

### Native copy and move

#### `copy(ctx, from, to, override)`

The runtime calls this only when `from` belongs to this Drive instance. `from` is a plain Entry (`Path`, `IsDir`, `Size`, `ModTime`, `Data`). Throw `ErrUnsupported()` when native copy is unavailable (for example directories). The dispatcher will fall back to reading the source and calling destination `save`. Never disguise an actual remote failure as Unsupported.

#### `move(ctx, from, to, override)`

Same ownership wrapping as `copy`. `ErrUnsupported()` from `move` does **not** trigger automatic copy-and-delete.

### Upload strategy

#### `upload(ctx, path, size, override, config) -> DriveUploadConfig | undefined`

Chooses the frontend upload strategy; it does not replace `save`. Defaults to `useLocalProvider(size)`.

- Return `useCustomProvider(safeConfig)` for direct browser uploads (no uploader name).
- After a successful browser upload the runtime calls this again with `config.action === "Completed"` and evicts the target and parent. Return immediately for that action unless the Drive must finish a server-side commit.
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

#### `getThumbnail(ctx, entry) -> ReadCloser | ContentURL` (optional)

Return a remote thumbnail response body or URL configuration. When returning the body, do not dispose it first. Mark eligible entries with `Meta.SelfThumbnail: true` in `get`/`list` (type, extension, size only; no network). Omit `getThumbnail` when the service has no thumbnail capability.

## 7. Available JavaScript APIs

The following runtime surface is safe to depend on. Refer to the two `.d.ts` files for exact field types.

### Configuration, state, and cache

- `utils.Config`: `OAuthRedirectURI`, `Version`, `RevHash`, and `BuildAt`.
- `utils.Data.Load(...keys)` / `utils.Data.Save(map)`: persistent string configuration.
- `this.cache`: entry cache created for the instance. Use it only for extra invalidation; `get`/`list` and write methods are wrapped automatically.
- `parseDuration(str)`: parse a Go duration string (`ms`/`s`/`m`/`h`).
- `DriveCache.PutEntry`, `PutEntries`, and `PutChildren`.
- `DriveCache.GetEntry` and `GetChildren`; a miss returns `null`.
- `DriveCache.Evict(path, descendants)` and `EvictAll()`.
- Cross-VM shared state: assign `$` properties on the instance (`this.$foo = …`).
- `selfDrive`: the Go wrapper of the current script Drive, with Get/Save/MakeDir/Copy/Move/List/Delete methods.

### OAuth

- `utils.OAuthInitConfig(request, credentials)`: produce a configuration/OAuth step and possibly an existing `OAuthHolder`.
- `utils.OAuthInit(ctx, data, request, credentials)`: handle the OAuth callback during initialization.
- `utils.OAuthLoad(request, credentials)`: construct the runtime `OAuthHolder` from a stored token.
- `OAuthHolder.Token(ctx)`: retrieve an automatically refreshed token. `ctx` is required; refresh uses it.
- An OAuth request contains Endpoint, RedirectURL, Scopes, and Text; credentials contain ClientID and ClientSecret.
- Endpoint authentication styles are `OAuthStyle.AutoDetect`, `InParams`, and `InHeader`. Prefer auto-detection unless the provider requires otherwise.

Follow `dropbox.js`. Do not persist OAuth state manually or duplicate refresh-token logic.

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

Supported types are `md`, `textarea`, `text`, `password`, `checkbox`, `checkboxes`, `select`, `path`, `form`, and `code`. Drive credentials normally need only text/password/select/checkbox. Use `Type: "password"` for secrets. When an existing secret is returned through `DriveInitConfiguration.Value`, the admin API replaces it with a reserved placeholder; submitting that unchanged placeholder preserves the stored secret.

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
// @name Example REST Drive
// @version 1.0.0
// @uploader example-uploader.js
// @description Example of a complete HTTP API based adapter.
//
// Enter the API endpoint and a token with file read/write permissions.

/// <reference path="../docs/scripts/env/drive.d.ts"/>

defineDrive(
  {
    configForm: [
      { Label: "API URL", Field: "base_url", Type: "text", Required: true },
      { Label: "Token", Field: "token", Type: "password", Required: true },
      entryCacheTTLFormItem("5m")
    ],

    validateConfig: function (config) {
      if (!/^https:\/\/[^/]+(?:\/.*)?$/.test(config.base_url || "")) {
        throw ErrBadRequest("API URL must use HTTPS");
      }
    },

    createInstance: function (ctx, config) {
      return {
        entryCacheTTL: config.cache_ttl,
        baseURL: config.base_url.replace(/\/+$/, ""),
        token: config.token
      };
    }
  },
  {
    get: function (ctx, path) {
      var result = requestJSON(
        this,
        ctx,
        "GET",
        "/v1/entries?path=" + encodeURIComponent(path)
      );
      return toEntry(result.entry);
    },

    list: function (ctx, path) {
      var all = [];
      var cursor = "";
      do {
        ctx.Err();
        var route = "/v1/children?path=" + encodeURIComponent(path);
        if (cursor) route += "&cursor=" + encodeURIComponent(cursor);
        var page = requestJSON(this, ctx, "GET", route);
        for (var i = 0; i < page.items.length; i++) {
          all.push(toEntry(page.items[i]));
        }
        cursor = page.nextCursor || "";
      } while (cursor);
      return all;
    },

    save: function (ctx, path, size, override, reader) {
      ctx.Total(size, true);
      var route = "/v1/content?path=" + encodeURIComponent(path) +
        "&override=" + (override ? "true" : "false");
      var resp = http(
        ctx,
        "PUT",
        this.baseURL + route,
        {
          Authorization: "Bearer " + this.token,
          "Content-Type": "application/octet-stream"
        },
        reader.ProgressReader(ctx)
      );
      var status = resp.Status;
      var message = resp.Text();
      if (status === 409) throw ErrNotAllowed("destination already exists");
      if (status < 200 || status >= 300) throw ErrRemoteApi(status, message);
    },

    makeDir: function (ctx, path) {
      requestJSON(this, ctx, "POST", "/v1/directories", { path: path });
    },

    copy: function (ctx, from, to, override) {
      throw ErrUnsupported();
    },

    move: function (ctx, from, to, override) {
      throw ErrUnsupported();
    },

    delete: function (ctx, path) {
      requestJSON(
        this,
        ctx,
        "DELETE",
        "/v1/entries?recursive=true&path=" + encodeURIComponent(path)
      );
    },

    getURL: function (ctx, entry) {
      var data = requestJSON(
        this,
        ctx,
        "GET",
        "/v1/download-url?path=" + encodeURIComponent(entry.Path)
      );
      return { URL: data.url };
    }
  }
);


function requestJSON(drive, ctx, method, route, body) {
  var headers = {
    Authorization: "Bearer " + drive.token,
    Accept: "application/json"
  };
  var payload;
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
    payload = JSON.stringify(body);
  }
  var resp = http(ctx, method, drive.baseURL + route, headers, payload);
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

function toEntry(remote) {
  return {
    IsDir: remote.type === "dir",
    Path: pathUtils.clean(remote.path),
    Size: remote.type === "dir" ? -1 : remote.size,
    ModTime: remote.modified_at ? dayjs(remote.modified_at).valueOf() : -1,
    Data: { id: String(remote.id) }
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
    var res = await ctx.request({ method: "post", url: ctx.config.initURL });
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
3. Define one unambiguous remote-object-to-Entry mapping, including root path, directory emulation, and time units.
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
11. Enable `GO_DRIVE_DEBUG=1` only for temporary diagnosis. Confirm logs are redacted, then disable it.
12. Do not edit `build/`, `web/dist/`, or dependency directories, and never commit real credentials.

Completion means every contract above is implemented or has explicit Unsupported behavior, and read/write operations, errors, cancellation, and resource ownership have been verified against a real test instance. Merely loading the script is not sufficient.

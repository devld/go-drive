## Adapting Drive with JavaScript

Create a `.js` file for the Drive. It runs in Goja with ES6+ syntax (`let`/`const`, arrow functions, classes, rest/spread, and modern array methods). Goja provides standard `Promise` objects, but all go-drive host APIs are synchronous and do not return Promises. There is no JavaScript event loop, timer API, or asynchronous HTTP API, and top-level `await` is not supported. The server runtime does not provide `import`, `export`, or `require`, DOM APIs, `fetch`, `XMLHttpRequest`, or Node.js APIs. Available APIs are in [`global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts) and [`drive.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/env/drive.d.ts).

The built-in go-drive globals and utility objects are immutable. After a pooled VM finishes initialization, its global object is frozen so Drive methods cannot add or replace globals. Scripts cannot replace or extend APIs such as `http`, `console`, `pathUtils`, `urlUtils`, or `dayjs`. Values produced by Go (`urlUtils.parse`, `resp.json()`, cache items, Drive `config`, `$event`, shared `$` reads, and similar) are read-only views: copy with object or array spread before editing.

The Goja migration is a breaking script API change. Go fields and methods exposed to JavaScript now use lowerCamel names (for example, `resp.Status` becomes `resp.status`, `resp.JSON()` becomes `resp.json()`, `entry.GetURL()` becomes `entry.getUrl()`, and `OAuthLoad` becomes `oauthLoad`). Existing Otto/ES5 scripts must be updated; there is no PascalCase compatibility layer.

Leading `// @name`, `// @version`, and optional `// @description` / `// @uploader` comments identify the script. Implement the adapter with `defineDrive(setup, methods)`.

See [`script-drives`](https://github.com/devld/go-drive/tree/master/script-drives) and [`script-drives/AGENTS.md`](https://github.com/devld/go-drive/blob/master/script-drives/AGENTS.md).

Copy the file into `/script-drives` when finished.

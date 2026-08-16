## Implementing Drive's uploader

JavaScript drives can upload through go-drive's local uploader (the file is relayed by the server).

For a direct browser upload, add a sibling uploader script and point to it with `// @uploader <filename>.js` on the Drive script. After install it is served as `/drive-uploader/<drive-name>`.

The uploader runs in the browser. Call `defineUploader({ ... })` with:

- optional `chunkSize` / `maxConcurrent` / `start` / `complete` / `abort` / `getChunk`
- required `upload(ctx, { blob, seq, session, onProgress })`

The runtime slices the file, aborts on failure/cancel, and notifies the Drive with `{ action: "Completed" }` after `complete`. Types: [`types.d.ts`](./types.d.ts). Example: [`script-drives/qiniu-uploader.js`](https://github.com/devld/go-drive/blob/master/script-drives/qiniu-uploader.js).

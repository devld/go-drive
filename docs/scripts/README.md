## Adapting Drive with JavaScript

Create a `.js` file for the Drive. It runs in Otto (**ES5** only, no DOM). Available APIs are in [`global.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/global.d.ts) and [`drive.d.ts`](https://github.com/devld/go-drive/blob/master/docs/scripts/env/drive.d.ts).

Leading `// @name`, `// @version`, and optional `// @description` / `// @uploader` comments identify the script. Implement the adapter with `defineDrive(setup, methods)`.

See [`script-drives`](https://github.com/devld/go-drive/tree/master/script-drives) and [`script-drives/AGENTS.md`](https://github.com/devld/go-drive/blob/master/script-drives/AGENTS.md).

Copy the file into `/script-drives` when finished.

# Frontend workspace packages

The web application remains the workspace root and the packages below are
installed together with the single `web/package-lock.json` file:

- `@go-drive/monaco-editor` contains the Monaco iframe, workers, protocol and
  language mapping.
- `@go-drive/code-mirror` contains the reusable CodeMirror Vue editor and
  lazy language extensions.
- `@go-drive/utils` contains shared browser/Vue utilities, types and loading
  components without go-drive application dependencies.
- `@go-drive/previewers` contains browser-only preview components and their
  format-specific dependencies.

The packages accept plain values and callbacks. Authentication, file access,
configuration, saving and application UI stay in `web/src`, so the packages
do not import go-drive application modules. Run `npm install` from `web` to
install the root application and all workspaces together.

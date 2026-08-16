# Unreleased

Copy this section into the next GitHub release notes.

## Breaking changes

### Script Drive metadata comments

Script Drive headers now use `@` directives only. The previous first-line display name, `version:` aliases, and free-form comment description are no longer parsed.

- `@name` and `@version` are required. Scripts without either are ignored by repository sync and the installed-script list.
- Optional `@uploader` and `@description` replace the old header fields.
- `@description` may span multiple following `//` lines that are not themselves `@` directives.
- Repository sync no longer treats `*-uploader.js` as a special filename. An uploader is attached only when the drive script declares `@uploader`.

Update existing third-party scripts before upgrading. The bundled Dropbox and Qiniu adapters already use the new header format.

### Script Drive API (`defineDrive` / `defineUploader`)

The previous `defineCreate` / `defineInitConfig` / `defineInit` hooks and factory-function uploaders are removed.

- Implement adapters with `defineDrive(setup, methods)`. `setup` covers the form, validation, OAuth, and `createInstance`; `methods` are the Drive operations. `this` is inferred from `createInstance`. `$` fields are shared JSON state across VMs.
- Entry cache TTL is an optional form field (`entryCacheTTLFormItem`). Pass the raw string as `entryCacheTTL` from `createInstance`. Cache get/put/evict, root `get("")`, write-path eviction, and copy/move ownership run in Go so read hits do not occupy the Otto pool. Existing instances without `cache_ttl` saved will not cache until the Drive config is saved again.
- Copy/move ownership uses the Drive instance, not `Entry.Data.d`. Put only remote ids in `Data`.
- `useCustomProvider(config)` no longer takes an uploader name.
- Browser uploaders must call `defineUploader`. The runtime handles chunking, abort, and the `Completed` callback.

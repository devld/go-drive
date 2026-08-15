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

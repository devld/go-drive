/// <reference path="./libs/dayjs.d.ts" />

/** Shared synchronous APIs for Drive and job scripts running on Goja. */

/** String map produced by Go; properties cannot be assigned. */
declare type SM = { readonly [key: string]: string };
/** Object map produced by Go; properties cannot be assigned. */
declare type M<T = any> = { readonly [key: string]: T };

declare type JSONPrimitive = string | number | boolean | null;
declare type ReadonlyJSONValue =
  | JSONPrimitive
  | readonly ReadonlyJSONValue[]
  | { readonly [key: string]: ReadonlyJSONValue };

declare const __go: unique symbol;
declare interface GoHandle<Id extends string> {
  readonly [__go]: Id;
}

/** Runtime console backed by go-drive logging. */
declare interface Console {
  debug(...message: any[]): void;
  error(...message: any[]): void;
  info(...message: any[]): void;
  log(...message: any[]): void;
  warn(...message: any[]): void;
}
declare var console: Console;

/** Block for `t`. Example: `sleep(ms(1000))` or `sleep("1s")`. */
declare function sleep(t: DurationLike): void;

declare type BytesEncoding = "utf8" | "hex" | "base64" | "base64url";

declare interface BytesEncodeOptions {
  /** For `base64` / `base64url`. Defaults to `true`. Pass `false` for JWT / PKCE. */
  padded?: boolean;
}

/**
 * Byte buffer. `new Bytes(n)` allocates `n` zeroed bytes (`n` is at most 32MiB).
 * `Bytes.fromString("hi")` copies UTF-8.
 */
declare class Bytes {
  constructor(length: number);
  static fromString(s: string): Bytes;
  static fromHex(s: string): Bytes;
  static fromBase64(s: string, options?: BytesEncodeOptions): Bytes;
  static fromBase64Url(s: string, options?: BytesEncodeOptions): Bytes;
  /** Cryptographically random bytes. `n` must be in `[0, 1MiB]`. */
  static random(n: number): Bytes;
  get length(): number;
  /** Slice `[start, end)`. */
  slice(start: number, end: number): Bytes;
  /**
   * Encode as text. `encoding` defaults to `utf8`.
   * `options.padded` applies to `base64` / `base64url` (default `true`).
   */
  toString(encoding?: BytesEncoding, options?: BytesEncodeOptions): string;
}

/**
 * Operation-scoped task progress capability. Cannot be constructed from JavaScript.
 * Derived reporters may only remove permissions, never add them.
 */
declare class ProgressReporter {
  addLoaded(delta: number): void;
  addTotal(delta: number): void;
  derive(capabilities: {
    loaded?: boolean;
    total?: boolean;
  }): ProgressReporter;
}

/** Go `io.Reader`. */
declare class Reader {
  /** Read into `dest` (up to `dest.length`). Returns bytes read, or `-1` at EOF. */
  read(dest: Bytes): number;
  readAsString(): string;
  /** Limit remaining reads to `n` bytes. */
  limitReader(n: number): Reader;
  /** Return a Reader that adds consumed bytes to `reporter.loaded`. */
  withProgress(reporter: ProgressReporter): Reader;
}

/** Go `io.ReadCloser`. */
declare class ReadCloser extends Reader {
  close(): void;
}

/** Seek from start of file. */
declare const SEEK_START = 0;
/** Seek from current offset. */
declare const SEEK_CURRENT = 1;
/** Seek from end of file. */
declare const SEEK_END = 2;

declare class TempFile extends ReadCloser {
  constructor();
  write(b: Bytes): void;
  copyFrom(r: Reader): void;
  /** Seek; `whence` is `SEEK_*`. Returns the new absolute offset. */
  seekTo(offset: number, whence: number): number;
  size(): number;
}

declare type HashAlgorithm = "md5" | "sha1" | "sha256" | "sha512";

declare class Hash {
  constructor(algo: HashAlgorithm);
  write(b: Bytes): this;
  /**
   * Hash `r` from the current offset to EOF. Does not close `r`.
   * Does not seek to the start. To hash a whole TempFile, `seekTo(0, SEEK_START)` first.
   * If `r` is seekable (`TempFile`), the original offset is restored afterwards.
   */
  writeFrom(r: Reader): this;
  sum(): Bytes;
}
declare class Hmac extends Hash {
  constructor(algo: HashAlgorithm, key: Bytes);
}

declare type EntryType = "file" | "dir";

declare interface EntryMeta {
  readonly readable: boolean;
  readonly writable: boolean;
  /** Client-loadable thumbnail URL. */
  readonly thumbnailUrl?: string;
  /**
   * Set when `getThumbnail` can produce a thumbnail for this entry.
   * Local predicate only (type, extension, size); no network I/O.
   */
  readonly selfThumbnail?: boolean;
  readonly props?: M;
}

declare interface ContentURL {
  readonly url: string;
  /** Extra request headers. */
  readonly header?: SM;
  /** Proxy the download through go-drive. */
  readonly proxy?: boolean;
  /** Content-Disposition filename; defaults to the entry name. */
  readonly downloadFileName?: string;
}

/**
 * Host Drive (`selfDrive` in Drive scripts, `drive` in jobs).
 * Cannot be constructed from JavaScript. Distinct from `DriveThis` (`defineDrive` `this`).
 */
declare class Drive {
  get(path: string): Entry;
  save(
    path: string,
    size: number,
    override: boolean,
    reader: Reader,
    progress: ProgressReporter
  ): Entry;
  makeDir(path: string): Entry;
  copy(
    from: Entry,
    to: string,
    override: boolean,
    progress: ProgressReporter
  ): Entry;
  move(
    from: Entry,
    to: string,
    override: boolean,
    progress: ProgressReporter
  ): Entry;
  list(path: string): readonly Entry[];
  delete(path: string, progress: ProgressReporter): void;
}

/**
 * Host entry from `Drive` (getters, not `EntryRecord` fields).
 * Cannot be constructed from JavaScript.
 */
declare class Entry {
  get path(): string;
  get name(): string;
  get type(): EntryType;
  /** Size in bytes, or `-1` if unknown. */
  get size(): number;
  get meta(): EntryMeta;
  /** Last modified, Unix milliseconds. */
  get modTime(): number;
  /** Underlying entry if this one is wrapped. */
  get unwrap(): Entry;
  get data(): SM | null;
  get drive(): Drive | null;
  /** Throws `UnsupportedError` if not available. */
  getUrl(): ContentURL;
  /** Range read. Throws `UnsupportedError` if not available. */
  getReader(start: number, size: number): ReadCloser;
  /** Shape used by `JSON.stringify`. */
  toJSON(): {
    path: string;
    name: string;
    type: EntryType;
    size: number;
    modTime: number;
    meta: EntryMeta;
  };
}

declare type HttpMethod =
  | "HEAD"
  | "GET"
  | "POST"
  | "PUT"
  | "DELETE"
  | "PATCH"
  | "OPTIONS";

declare type HttpBody = Reader | string | Bytes | HttpFormData;

/** HTTP response headers. */
declare interface HttpHeaders extends GoHandle<"HttpHeaders"> {
  get(key: string): string;
  values(key: string): readonly string[];
  getAll(): { readonly [key: string]: readonly string[] };
}

/**
 * multipart/form-data body.
 * String and Bytes parts may be sent more than once. A form that includes a
 * Reader may be passed to `http()` only once. `HttpFormData` does not close
 * Readers; the caller closes `ReadCloser` / `TempFile`.
 */
declare class HttpFormData {
  constructor();
  appendField(key: string, data: string | Bytes): void;
  appendFile(
    key: string,
    filename: string,
    data: string | Bytes | Reader
  ): void;
}

/** Must call `dispose()` after use (unless `text()` or `json()` already did). */
declare interface HttpResponse extends GoHandle<"HttpResponse"> {
  readonly status: number;
  readonly headers: HttpHeaders;
  readonly body: ReadCloser;
  /** `Content-Length`, or `-1` if missing. */
  bodySize(): number;
  /** Read body as string and dispose this response. */
  text(): string;
  /** Parse the body as JSON and dispose this response; an empty body returns null. */
  json(): ReadonlyJSONValue | null;
  dispose(): void;
}

declare interface HttpRequestOptions {
  /** Defaults to `"GET"`. */
  method?: HttpMethod;
  headers?: SM;
  body?: HttpBody;
  /**
   * Per-request timeout. Defaults to `"30s"`.
   * `0` disables the extra timeout (the VM run context still applies).
   * Other host network APIs do not add a timeout.
   */
  timeout?: DurationLike;
}

/**
 * HTTP request. Dispose the response when finished.
 *
 * For a Reader `body`, `Content-Length` is taken from `headers` when present
 * (the body is truncated to that size). Otherwise a known size is used
 * (`TempFile` remaining bytes, including `limitReader` wrapping one).
 * String and Bytes always use their actual length. HttpFormData is
 * multipart. Set `Transfer-Encoding: chunked` to skip auto `Content-Length`.
 * Progress is explicit: wrap a Reader body with `reader.withProgress(reporter)`.
 */
declare function http(url: string, req?: HttpRequestOptions): HttpResponse;

declare type FormItemType =
  | "md"
  | "textarea"
  | "text"
  | "password"
  | "checkbox"
  | "checkboxes"
  | "select"
  | "path"
  | "form"
  | "code";

declare interface FormItemOption {
  name: string;
  title?: string;
  value: string;
  disabled?: boolean;
}

declare interface FormItemPathOptions {
  /** Comma-separated picker filter (`file`, `dir`, `.ext`, `write`, `<size`). */
  filter?: string;
}

declare interface FormItemForm {
  key: string;
  name: string;
  form: FormItem[];
}

declare interface FormItemForms {
  addText?: string;
  maxItems?: number;
  forms: FormItemForm[];
}

declare interface FormItemCode {
  /** Highlight language. */
  type: string;
  typeSelectable?: boolean;
}

declare interface FormItem {
  label?: string;
  type: FormItemType;
  field: string;
  required?: boolean;
  /** Help text; markdown when `type` is `md`. */
  description?: string;
  disabled?: boolean;
  /** `select` */
  options?: FormItemOption[];
  /** `path` */
  pathOptions?: FormItemPathOptions;
  /** `form` */
  forms?: FormItemForms;
  /** `code` */
  code?: FormItemCode;
  defaultValue?: string;
}

/** Go `time.Duration` as nanoseconds. Build with `ms()`. */
declare type Duration = number;

/**
 * `ms(500)` or a Go duration string (`"2s"`, `"1h30m"`, `"2d3h4m5s"`).
 * Days (`d`) are a go-drive extension; the rest is `time.ParseDuration`.
 */
declare type DurationLike = Duration | string;

/** Milliseconds → Go duration. */
declare function ms(ms: number): Duration;

/** Parse `ms(500)` or a duration string. Empty / omitted → `0`. Invalid throws TypeError. */
declare function parseDuration(value?: DurationLike): Duration;

/** HTTP 400. */
declare class BadRequestError extends Error {
  constructor(msg?: string);
}

/** HTTP 404. */
declare class NotFoundError extends Error {
  constructor(msg?: string);
}

/** HTTP 403. */
declare class NotAllowedError extends Error {
  constructor(msg?: string);
}

declare class UnsupportedError extends Error {
  constructor(msg?: string);
}

/** Remote HTTP error with status. */
declare class RemoteApiError extends Error {
  constructor(status: number, msg: string);
  status: number;
}

/** POSIX-like path helpers (`/` separator, no leading slash). */
declare const pathUtils: {
  clean: (path: string) => string;
  join: (...segments: string[]) => string;
  parent: (path: string) => string;
  base: (path: string) => string;
  /** Lower-case extension without the dot. */
  ext: (path: string) => string;
  isRoot: (path: string) => boolean;
};

/** URL query parameters; each array preserves repeated query keys. Read-only when returned by `parse*`. */
declare type URLSearchParamsMap = { readonly [key: string]: readonly string[] };

/** URL components returned by `urlUtils.parse`. Read-only; copy before editing for `build`. */
declare interface URLParts {
  readonly origin: string;
  /** Scheme with the trailing colon, e.g. `https:`. */
  readonly protocol: string;
  readonly username: string;
  readonly password: string;
  /** Hostname plus port, without user information. */
  readonly host: string;
  readonly hostname: string;
  readonly port: string;
  /** Escaped path, including the leading slash when present. */
  readonly pathname: string;
  /** Query values; repeated keys are represented by multiple array items. */
  readonly searchParams: URLSearchParamsMap;
  /** Escaped fragment including `#`, or an empty string. */
  readonly hash: string;
}

/** Parse and build URLs using Go's `net/url` semantics. Parsed objects are read-only. */
declare const urlUtils: {
  parse: (url: string) => URLParts;
  build: (parts: Partial<URLParts>) => string;
  /** Parse a query string with an optional leading `?`. */
  parseSearchParams: (search: string) => URLSearchParamsMap;
  /** Build a query string with a leading `?`, or an empty string. */
  buildSearchParams: (searchParams: URLSearchParamsMap) => string;
};

/** Editable tree node. Entry handles themselves remain read-only. */
declare interface EntryTreeNode {
  entry: Entry;
  children?: EntryTreeNode[];
  /** Skip this node in `flattenEntriesTree`; its children are still visited. */
  excluded?: boolean;
}

/** Walk a directory tree. `byteProgress` reports size instead of entry count. */
declare function buildEntriesTree(
  entry: Entry,
  byteProgress?: boolean,
  progress?: ProgressReporter
): EntryTreeNode;

/** Glob under `root`. */
declare function findEntries(
  root: Drive,
  pattern: string,
  bytesProgress?: boolean,
  progress?: ProgressReporter
): readonly Entry[];

/** Flatten a tree. `deepFirst` visits children before the node. */
declare function flattenEntriesTree(
  node: EntryTreeNode,
  deepFirst?: boolean
): EntryTreeNode[];

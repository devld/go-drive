/// <reference path="./libs/dayjs.d.ts" />

/** Shared APIs for Drive and job scripts (Otto ES5). */

/** Debug mode. */
declare const DEBUG: boolean;

/** String map. */
declare type SM = { [key: string]: string };
/** Object map. */
declare type M<T = any> = { [key: string]: T };

/** Write to the process console. */
declare function consoleWrite(level: string, ...msg: any[]): void;

/** Block for `t`. Example: `sleep(ms(1000))`. */
declare function sleep(t: Duration): void;

/** New Context (`context.Background()`). */
declare function newContext(): Context;

/**
 * Child Context cancelled after `timeout`.
 * Call `Cancel()` when finished (success or failure).
 */
declare function newContextWithTimeout(
  parent: Context,
  timeout: Duration
): ContextWithTimeout;

/** Task progress callback: `(loaded, total)`. */
declare type TaskCtxOnUpdate = (loaded: number, total: number) => void;
/** Wrap `ctx` as a `TaskCtx`. */
declare function newTaskCtx(ctx: Context, onUpdate?: TaskCtxOnUpdate): TaskCtx;

/** Mutex. Always `Unlock` in `finally`. */
declare function newLocker(): Locker;

/** Go context. */
declare interface Context {
  /** Throw if cancelled or timed out. */
  Err(): void;
}

declare interface ContextWithTimeout extends Context {
  /** Must be called when this Context is no longer used. */
  Cancel(): void;
}

/** Context that can report progress. */
declare interface TaskCtx extends Context {
  /** Report loaded bytes. `abs`: absolute vs delta. */
  Progress(loaded: number, abs: boolean): void;
  /** Report total bytes. `abs`: absolute vs delta. */
  Total(total: number, abs: boolean): void;
}

declare interface Locker {
  Lock(): void;
  Unlock(): void;
}

/** Bytes from a string. */
declare function newBytes(s: string): Bytes;
/** Zeroed Bytes of length `n`. */
declare function newEmptyBytes(n: number): Bytes;
/** Temp file; deleted on `Close()`. */
declare function newTempFile(): TempFile;

declare interface Bytes {
  Len(): number;
  /** Slice `[start, end)`. */
  Slice(start: number, end: number): Bytes;
  String(): string;
}

/** Go `io.Reader`. */
declare interface Reader {
  /** Read into `dest` (up to `dest.Len()`). Returns bytes read, or `-1` at EOF. */
  Read(dest: Bytes): number;
  ReadAsString(): string;
  /** Limit remaining reads to `n` bytes. */
  LimitReader(n: number): Reader;
  /** Report read progress to `ctx`. */
  ProgressReader(ctx: TaskCtx): Reader;
}

/** Go `io.ReadCloser`. */
declare interface ReadCloser extends Reader {
  Close(): void;
}

/** Seek from start of file. */
declare const SEEK_START = 0;
/** Seek from current offset. */
declare const SEEK_CURRENT = 1;
/** Seek from end of file. */
declare const SEEK_END = 2;

declare interface TempFile extends ReadCloser {
  Write(b: Bytes): void;
  CopyFrom(r: Reader): void;
  /** Seek; `whence` is `SEEK_*`. Returns the new absolute offset. */
  SeekTo(offset: number, whence: number): number;
  Size(): number;
}

declare type EntryType = "file" | "dir";

declare interface EntryMeta {
  Readable: boolean;
  Writable: boolean;
  /** Client-loadable thumbnail URL. */
  ThumbnailURL?: string;
  /**
   * Set when `getThumbnail` can produce a thumbnail for this entry.
   * Local predicate only (type, extension, size); no network I/O.
   */
  SelfThumbnail?: boolean;
  Props?: M;
}

declare interface ContentURL {
  URL: string;
  /** Extra request headers. */
  Header?: SM;
  /** Proxy the download through go-drive. */
  Proxy?: boolean;
  /** Content-Disposition filename; defaults to the entry name. */
  DownloadFileName?: string;
}

/** Host Drive API (root Drive in jobs, or `selfDrive` in Drive scripts). */
declare interface DriveInstance {
  Get(ctx: Context, path: string): DriveEntry;
  Save(
    ctx: TaskCtx,
    path: string,
    size: number,
    override: boolean,
    reader: Reader
  ): DriveEntry;
  MakeDir(ctx: Context, path: string): DriveEntry;
  Copy(
    ctx: TaskCtx,
    from: DriveEntry,
    to: string,
    override: boolean
  ): DriveEntry;
  Move(
    ctx: TaskCtx,
    from: DriveEntry,
    to: string,
    override: boolean
  ): DriveEntry;
  List(ctx: Context, path: string): DriveEntry[];
  Delete(ctx: TaskCtx, path: string): void;
}

declare interface DriveEntry {
  Path(): string;
  Name(): string;
  Type(): EntryType;
  /** Size in bytes, or `-1` if unknown. */
  Size(): number;
  Meta(): EntryMeta;
  /** Last modified, Unix milliseconds. */
  ModTime(): number;
  /** Throws `ErrUnsupported` if not available. */
  GetURL(ctx: Context): ContentURL;
  /** Range read. Throws `ErrUnsupported` if not available. */
  GetReader(ctx: Context, start: number, size: number): ReadCloser;
  /** Underlying entry if this one is wrapped. */
  Unwrap(): DriveEntry;
  Data(): SM | null;
  Drive(): DriveInstance;
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
declare interface HttpHeaders {
  Get(key: string): string;
  Values(key: string): string[] | null;
  GetAll(): M<string[]>;
}

/** multipart/form-data body. */
declare interface HttpFormData {
  AppendField(key: string, data: string | Bytes): void;
  AppendFile(
    key: string,
    filename: string,
    data: string | Bytes | Reader
  ): void;
}

/** Must call `Dispose()` after use (unless `Text()` already did). */
declare interface HttpResponse {
  Status: number;
  Headers: HttpHeaders;
  Body: ReadCloser;
  /** `Content-Length`, or `-1` if missing. */
  BodySize(): number;
  /** Read body as string and dispose this response. */
  Text(): string;
  Dispose(): void;
}

declare function newFormData(): HttpFormData;

declare interface HttpRequestOptions {
  headers?: SM;
  body?: HttpBody;
}

/**
 * HTTP request. Dispose the response when finished.
 *
 * For a Reader `body`, `Content-Length` is taken from `headers` when present
 * (the body is truncated to that size). Otherwise a known size is used
 * (`TempFile` remaining bytes, including `ProgressReader` / `LimitReader`
 * wrapping one). String and Bytes always use their actual length. FormData is
 * multipart. Set `Transfer-Encoding: chunked` to skip auto `Content-Length`.
 */
declare function http(
  ctx: Context,
  method: HttpMethod,
  url: string,
  req?: HttpRequestOptions
): HttpResponse;

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
  Name: string;
  Title?: string;
  Value: string;
  Disabled?: boolean;
}

declare interface FormItemPathOptions {
  /** Comma-separated picker filter (`file`, `dir`, `.ext`, `write`, `<size`). */
  Filter?: string;
}

declare interface FormItemForm {
  Key: string;
  Name: string;
  Form: FormItem[];
}

declare interface FormItemForms {
  AddText?: string;
  MaxItems?: number;
  Forms: FormItemForm[];
}

declare interface FormItemCode {
  /** Highlight language. */
  Type: string;
  TypeSelectable?: boolean;
}

declare interface FormItem {
  Label?: string;
  Type: FormItemType;
  Field: string;
  Required?: boolean;
  /** Help text; markdown when `Type` is `md`. */
  Description?: string;
  Disabled?: boolean;
  /** `select` */
  Options?: FormItemOption[];
  /** `path` */
  PathOptions?: FormItemPathOptions;
  /** `form` */
  Forms?: FormItemForms;
  /** `code` */
  Code?: FormItemCode;
  DefaultValue?: string;
}

/** Go `time.Time`. Convert with `dayjs(t.UnixMilli())`. */
declare interface GoTime {
  UnixMilli(): number;
}

/** Go `time.Duration`. Build with `ms()`. */
declare type Duration = number;

/** Milliseconds → Go duration. */
declare function ms(ms: number): Duration;

/**
 * Parse a Go duration (`ms`/`s`/`m`/`h`, optional `d` prefix).
 * Empty → `0`. Invalid → `-1`.
 */
declare function parseDuration(value?: string): Duration;

declare function toDate(goTime: GoTime): Date;

/** HTTP 400. */
declare function ErrBadRequest(msg?: string): Error;
declare function isBadRequestErr(e: any): boolean;

/** HTTP 404. */
declare function ErrNotFound(msg?: string): Error;
declare function isNotFoundErr(e: any): boolean;

/** HTTP 403. */
declare function ErrNotAllowed(msg?: string): Error;
declare function isNotAllowedErr(e: any): boolean;

declare function ErrUnsupported(msg?: string): Error;
declare function isUnsupportedErr(e: any): boolean;

/** Remote HTTP error with status. */
declare function ErrRemoteApi(status: number, msg: string): Error;
declare function isRemoteApiErr(e: any): boolean;

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

/** Hash algorithm for `encUtils`. */
declare enum HASH {
  MD5 = 1,
  SHA1 = 2,
  SHA256 = 3,
  SHA512 = 4,
}

declare interface Hasher {
  Write(b: Bytes): Hasher;
  /**
   * Hash `r` from the current offset to EOF. Does not close `r`.
   * Does not seek to the start. To hash a whole TempFile, `SeekTo(0, SEEK_START)` first.
   * If `r` is seekable (`TempFile`), the original offset is restored afterwards.
   */
  WriteReader(r: Reader): Hasher;
  Sum(): Bytes;
}

/** Hex / Base64 / HMAC / digest helpers. */
declare const encUtils: {
  toHex: (b: Bytes) => string;
  fromHex: (s: string) => Bytes;
  /** `padded` defaults to `true`. Pass `false` for raw (no `=`). */
  base64Encode: (b: Bytes, padded?: boolean) => string;
  base64Decode: (s: string, padded?: boolean) => Bytes;
  /** URL-safe Base64. `padded` defaults to `true`; `false` for JWT / PKCE. */
  urlBase64Encode: (b: Bytes, padded?: boolean) => string;
  urlBase64Decode: (s: string, padded?: boolean) => Bytes;
  /** Cryptographically random bytes. `n` must be in `[0, 1MiB]`. */
  randomBytes: (n: number) => Bytes;
  newHash: (h: HASH) => Hasher;
  /** Streaming HMAC; `Write` / `WriteReader` / `Sum` like `newHash`. */
  newHmac: (h: HASH, key: Bytes) => Hasher;
};

declare interface EntryTreeNode {
  Entry: DriveEntry;
  Children?: EntryTreeNode[];
  /** Filter miss; skipped by `flattenEntriesTree`. */
  Excluded?: boolean;
}

/** Walk a directory tree. `byteProgress` reports size instead of entry count. */
declare function buildEntriesTree(
  ctx: TaskCtx,
  entry: DriveEntry,
  byteProgress?: boolean
): EntryTreeNode;

/** Glob under `root`. */
declare function findEntries(
  ctx: TaskCtx,
  root: DriveInstance,
  pattern: string,
  bytesProgress?: boolean
): DriveEntry[];

/** Flatten a tree. `deepFirst` visits children before the node. */
declare function flattenEntriesTree(
  node: EntryTreeNode,
  deepFirst?: boolean
): EntryTreeNode[];

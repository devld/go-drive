/// <reference path="./libs/dayjs.d.ts" />

/** is debug mode on */
declare const DEBUG: boolean;

/** String Map */
declare type SM = { [key: string]: string };
/** Map */
declare type M<T = any> = { [key: string]: T };

/**
 * write something to console
 * @param level string
 * @param msg messages
 */
declare function consoleWrite(level: string, ...msg: string[]): void;

/**
 * Pause for a while
 *
 * Example: `sleep(ms(1000))`
 *
 * @param t duration
 */
declare function sleep(t: Duration): void;

/** Creates a new Context */
declare function newContext(): Context;

/** Wraps a Context and cancels it after the timeout.
 *
 * Example: `newContextWithTimeout(newContext(), ms(30 * 1000))`
 *
 * @param parent parent context
 * @param timeout timeout
 * @returns **`Cancel` must be called at the function ends**
 */
declare function newContextWithTimeout(
  parent: Context,
  timeout: Duration
): ContextWithTimeout;

/** The progress callback of TaskCtx */
declare type TaskCtxOnUpdate = (loaded: number, total: number) => void;
/**
 *
 * @param ctx wrapped Context
 * @param onUpdate on progress update callback
 */
declare function newTaskCtx(ctx: Context, onUpdate?: TaskCtxOnUpdate): TaskCtx;

/** create a locker */
declare function newLocker(): Locker;

/** Context of Go */
declare interface Context {
  /** Detects error of this Context, such as whether it was cancelled, timed out, etc. Any errors will be thrown */
  Err(): void;
}

declare interface ContextWithTimeout extends Context {
  /** Cancel this Context. Cancel **MUST** be called at the end of the Context's use (whether successful or unsuccessful) */
  Cancel(): void;
}

/** Context wrapper. Used to report operation progress */
declare interface TaskCtx extends Context {
  /**
   * Report progress
   * @param abs is absolute value
   */
  Progress(loaded: number, abs: boolean): void;
  /**
   * Report progress total
   * @param abs is absolute value
   */
  Total(total: number, abs: boolean): void;
}

declare interface Locker {
  Lock(): void;
  Unlock(): void;
}

/**
 * Create a Bytes from string
 * @param s content
 */
declare function newBytes(s: string): Bytes;
/**
 * Create an empty Bytes
 * @param n size
 */
declare function newEmptyBytes(n: number): Bytes;
/**
 * Creates a temporary file that can be used for reading and writing. It will be deleted after closing
 */
declare function newTempFile(): TempFile;

declare interface Bytes {
  /**
   * Returns the size of this Bytes
   */
  Len(): number;
  /**
   * Create a Bytes slice from this Bytes
   * @param start start position
   * @param end end position(exclusion)
   */
  Slice(start: number, end: number): Bytes;
  /**
   * Converts this Bytes to string
   */
  String(): string;
}

/** Wrapper of Go io.Reader */
declare interface Reader {
  /**
   * Read contents into dest Bytes. It reads up to `dest.Len()`.
   * @param dest the data will be read into
   * @returns how much data has been read, returns `-1` if no more data
   */
  Read(dest: Bytes): number;
  /**
   * Read the whole data as string
   */
  ReadAsString(): string;
  /**
   * Creates a Reader that is limited to reading `n` data.
   *
   * Typically used to slice Reader
   */
  LimitReader(n: number): Reader;
  /**
   * Wraps this Reader and reports the progress to `ctx` when it is read
   */
  ProgressReader(ctx: TaskCtx): Reader;
}

/** Wrapper of Go io.ReadCloser */
declare interface ReadCloser extends Reader {
  /** Close this Reader */
  Close(): void;
}

/** seek relative to the origin of the file */
declare const SEEK_START = 0;
/** seek relative to the current offset */
declare const SEEK_CURRENT = 1;
/** seek relative to the end */
declare const SEEK_END = 2;

declare interface TempFile extends ReadCloser {
  /** Writes data into this file */
  Write(b: Bytes): void;
  /** Copy data from `r` into this file */
  CopyFrom(r: Reader): void;
  /**
   * Seek sets the offset for the next Read or Write on file to offset, interpreted according to whence:
   * - `SEEK_START`(`0`) means relative to the origin of the file
   * - `SEEK_CURRENT`(`1`) means relative to the current offset
   * - `SEEK_END`(`2`) means relative to the end.
   *
   * It returns the new absolute offset.
   */
  SeekTo(offset: number, whence: number): number;
}

/** The EntryType is `file` or `dir` */
declare type EntryType = "file" | "dir";

declare interface EntryMeta {
  Readable: boolean;
  Writable: boolean;
  Thumbnail?: string;
  Props?: M;
}

declare interface ContentURL {
  /** The URL */
  URL: string;
  /** The Headers passed to when sending request */
  Header?: SM;
  /** Is this request have to go through the server proxy */
  Proxy?: boolean;
}

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
  /** Returns the size of this entry. Returns `-1` if not available */
  Size(): number;
  Meta(): EntryMeta;
  /** Last modification time, in milliseconds */
  ModTime(): number;
  /** Get the entry's download URL. It throws `ErrUnsupported` when not supported */
  GetURL(ctx: Context): ContentURL;
  /** Get the Reader of the entry's content. It throws `ErrUnsupported` when not supported */
  GetReader(ctx: Context, start: number, size: number): ReadCloser;
  /** Returns the wrapped real Entry */
  Unwrap(): DriveEntry;
  /** Returns the cached data of this entry */
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

declare interface HttpHeaders {
  Get(key: string): string;
  Values(key: string): string[] | null;
  GetAll(): M<string[]>;
}

declare interface HttpFormData {
  AppendField(key: string, data: string | Bytes): void;
  AppendFile(
    key: string,
    filename: string,
    data: string | Bytes | Reader
  ): void;
}

/** HttpResponse must be Disposed after use */
declare interface HttpResponse {
  Status: number;
  Headers: HttpHeaders;
  Body: ReadCloser;
  /** Returns the `Content-Length`, returns `-1` if no `Content-Length` */
  BodySize(): number;
  /** Read the whole body as string. `HttpResponse.Text` will dispose this HttpResponse */
  Text(): string;
  Dispose(): void;
}

/** Creates a FormData for http request */
declare function newFormData(): HttpFormData;

/** Sends a HTTP request */
declare function http(
  ctx: Context,
  method: HttpMethod,
  url: string,
  headers?: SM,
  body?: HttpBody
): HttpResponse;

declare type FormItemType =
  | "md"
  | "textarea"
  | "text"
  | "password"
  | "checkbox"
  | "select"
  | "form";

declare interface FormItemOption {
  Name: string;
  Title?: string;
  Value: string;
  Disabled?: boolean;
}

declare interface FormItemPathOptions {
  Filter?: string;
}

declare interface FormItemForm {
  Key: string;
  /** Display name */
  Name: string;
  Form: FormItem[];
}

declare interface FormItemForms {
  /** The display text of the add button */
  AddText?: string;
  /** The maximum items count can be added */
  MaxItems?: number;
  Forms: FormItemForm[];
}

declare interface FormItem {
  Label?: string;
  Type: FormItemType;
  Field: string;
  Required?: boolean;
  Description?: string;
  Disabled?: boolean;

  /** for FormItemType select */
  Options?: FormItemOption[];

  /** for FormItemType path */
  PathOptions?: FormItemPathOptions;

  /** for FormItemType form */
  Forms?: FormItemForms;

  DefaultValue?: string;
}

/** time.Time of Go. Use `dayjs(time.UnixMilli())` to convert this */
declare interface GoTime {
  UnixMilli(): number;
}

/** Use ms to create Duration */
declare type Duration = number;

/** millisecond to time.Duration of Go */
declare function ms(ms: number): Duration;

/** time.Duration of Go to Date */
declare function toDate(goTime: GoTime): Date;

/**
 * create a BadRequestError(400)
 */
declare function ErrBadRequest(msg?: string): Error;
declare function isBadRequestErr(e: any): boolean;

/**
 * create a NotFoundError(404)
 */
declare function ErrNotFound(msg?: string): Error;
declare function isNotFoundErr(e: any): boolean;

/**
 * create a NotAllowedError(403)
 */
declare function ErrNotAllowed(msg?: string): Error;
declare function isNotAllowedErr(e: any): boolean;

/**
 * create an UnsupportedError
 */
declare function ErrUnsupported(msg?: string): Error;
declare function isUnsupportedErr(e: any): boolean;

/**
 * create a RemoteApiError
 */
declare function ErrRemoteApi(status: number, msg: string): Error;
declare function isRemoteApiErr(e: any): boolean;

declare const pathUtils: {
  clean: (path: string) => string;
  join: (...segments: string[]) => string;
  parent: (path: string) => string;
  base: (path: string) => string;
  /** returns lower-case file extension name */
  ext: (path: string) => string;
  isRoot: (path: string) => boolean;
};

declare enum HASH {
  MD5 = 1,
  SHA1 = 2,
  SHA256 = 3,
  SHA512 = 4,
}

declare interface Hasher {
  Write(b: Bytes): Hasher;
  Sum(): Bytes;
}

declare const encUtils: {
  toHex: (b: Bytes) => string;
  fromHex: (s: string) => Bytes;
  base64Encode: (b: Bytes) => string;
  base64Decode: (s: string) => Bytes;
  urlBase64Encode: (b: Bytes) => string;
  urlBase64Decode: (s: string) => Bytes;
  newHash: (h: HASH) => Hasher;
  hmac: (h: HASH, payload: Bytes, key: Bytes) => Bytes;
};

declare interface EntryTreeNode {
  Entry: DriveEntry;
  Children?: EntryTreeNode[];
  Excluded?: boolean;
}

declare function buildEntriesTree(
  ctx: TaskCtx,
  entry: DriveEntry,
  byteProgress?: boolean
): EntryTreeNode;

declare function findEntries(
  ctx: TaskCtx,
  root: DriveInstance,
  pattern: string,
  bytesProgress?: boolean
): DriveEntry[];

declare function flattenEntriesTree(
  node: EntryTreeNode,
  deepFirst?: boolean
): EntryTreeNode[];

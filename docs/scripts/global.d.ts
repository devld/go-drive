/// <reference path="./libs/dayjs.d.ts" />

/** is debug mode on */
declare const DEBUG: boolean;

/** String Map */
declare type SM = { [key: string]: string };
/** Map */
declare type M<T = any> = { [key: string]: T };

declare function sleep(t: Duration): void;

declare function newContext(): Context;
declare function newContextWithTimeout(
  parent: Context,
  timeout: Duration
): ContextWithTimeout;
declare type TaskCtxOnUpdate = (loaded: number, total: number) => void;
declare function newTaskCtx(ctx: Context, onUpdate?: TaskCtxOnUpdate): TaskCtx;

/** Context of Go */
declare interface Context {
  Err(): void;
}

declare interface ContextWithTimeout extends Context {
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

declare function newBytes(s: string): Bytes;
declare function newEmptyBytes(n: number): Bytes;
declare function newTempFile(): TempFile;

declare interface Bytes {
  Len(): number;
  Slice(start: number, end: number): Bytes;
  String(): string;
}

/** Wrapper of Go io.Reader */
declare interface Reader {
  Read(dest: Bytes): number;
  ReadAsString(): string;
  LimitReader(n: number): Reader;
  ProgressReader(ctx: TaskCtx): Reader;
}

/** Wrapper of Go io.ReadCloser */
declare interface ReadCloser extends Reader {
  Close(): void;
}

/** seek relative to the origin of the file */
declare const SEEK_START = 0;
/** seek relative to the current offset */
declare const SEEK_CURRENT = 1;
/** seek relative to the end */
declare const SEEK_END = 2;

declare interface TempFile extends ReadCloser {
  Write(b: Bytes): void;
  CopyFrom(r: Reader): void;
  SeekTo(offset: number, whence: number): number;
}

declare type EntryType = "file" | "dir";

declare interface EntryMeta {
  Readable: boolean;
  Writable: boolean;
  Thumbnail?: string;
  Props?: M;
}

declare interface ContentURL {
  URL: string;
  Header?: SM;
  Proxy?: boolean;
}

declare interface RootDrive {
  Get(): DriveInstance;
  ReloadDrive(ctx: Context, ignoreFailure: boolean): void;
  ReloadMounts(): void;
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
  Size(): number;
  Meta(): EntryMeta;
  ModTime(): number;
  GetURL(ctx: Context): ContentURL;
  GetReader(ctx: Context): ReadCloser;
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
  BodySize(): number;
  /** Read the whole body as string. HttpResponse.Text will dispose this HttpResponse */
  Text(): string;
  Dispose(): void;
}

declare function newFormData(): HttpFormData;

declare function http(
  ctx: Context,
  method: HttpMethod,
  url: string,
  headers?: SM,
  body?: HttpBody
): HttpResponse;

declare type FormItemType =
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

declare interface FormItem {
  Label?: string;
  Type: FormItemType;
  Field: string;
  Required?: boolean;
  Description?: string;
  Disabled?: boolean;

  /** for FormItemType select */
  Options?: FormItemOption[];

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
  entry: DriveEntry;
  children?: EntryTreeNode[];
}

declare function buildEntriesTree(
  ctx: TaskCtx,
  entry: DriveEntry,
  byteProgress?: boolean
): EntryTreeNode;

declare function flattenEntriesTree(
  node: EntryTreeNode,
  result?: EntryTreeNode[]
): EntryTreeNode[];

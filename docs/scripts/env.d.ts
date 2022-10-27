/// <reference path="./dayjs/index.d.ts" />

/** is debug mode on */
declare const DEBUG: boolean;

/** String Map */
declare type SM = { [key: string]: string };
/** Map */
declare type M<T = any> = { [key: string]: T };

declare function newContext(): Context;
declare function newContextWithTimeout(
  parent: Context,
  timeout: Duration
): ContextWithTimeout;

/** Context of Go */
declare interface Context {}

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

declare interface DriveEntry {
  Path(): string;
  Name(): string;
  Type(): EntryType;
  Meta(): EntryMeta;
  ModTime(): number;
  GetURL(ctx: Context): ContentURL;
  GetReader(ctx: Context): ReadCloser;
  Unwrap(): DriveEntry;
  Data(): SM | null;
}

declare type HttpMethod =
  | "HEAD"
  | "GET"
  | "POST"
  | "PUT"
  | "DELETE"
  | "PATCH"
  | "OPTIONS";

declare type HttpBody = Reader | string;

declare interface HttpHeaders {
  Get(key: string): string;
  Values(key: string): string[] | null;
  GetAll(): M<string[]>;
}

/** HttpResponse must be Disposed after use */
declare interface HttpResponse {
  Status: number;
  Headers: HttpHeaders;
  Body: ReadCloser;
  /** Read the whole body as string. HttpResponse.Text will dispose this HttpResponse */
  Text(): string;
  Dispose(): void;
}

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

/**
 * create a NotFoundError(404)
 */
declare function ErrNotFound(msg?: string): Error;

/**
 * create a NotAllowedError(403)
 */
declare function ErrNotAllowed(msg?: string): Error;

/**
 * create an UnsupportedError
 */
declare function ErrUnsupported(msg?: string): Error;

/**
 * create a RemoteApiError
 */
declare function ErrRemoteApi(status: number, msg: string): Error;

declare const pathUtils: {
  clean: (path: string) => string;
  join: (...segments: string[]) => string;
  parent: (path: string) => string;
  base: (path: string) => string;
  /** returns lower-case file extension name */
  ext: (path: string) => string;
  isRoot: (path: string) => boolean;
};

declare const HASH_MD5: number;
declare const HASH_SHA1: number;
declare const HASH_SHA256: number;
declare const HASH_SHA512: number;

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

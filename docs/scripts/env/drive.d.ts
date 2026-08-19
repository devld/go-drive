/// <reference path="../global.d.ts"/>

declare const selfDrive: DriveInstance;

declare type JSONPrimitive = string | number | boolean | null;
declare type JSONValue =
  | JSONPrimitive
  | JSONValue[]
  | { [key: string]: JSONValue };

/** set drive's instance data */
declare function __setData(d: M<JSONValue>): void;
/** get drive's instance data by key */
declare function __getData(key: string): JSONValue;

declare interface DriveMeta {
  Writable: boolean;
  Props?: M;
}

/** Entry data between JavaScript and Go */
declare interface Entry {
  Meta?: EntryMeta;
  IsDir: boolean;
  Path: string;
  Size: number;
  ModTime: number;
  Data?: SM;
}

declare interface DriveUploadConfig {
  Provider: string;
  Path?: string;
  Config?: SM;
}

/** Drive interface that should be implemented */
declare interface Drive {
  cache: DriveCache;
  /** `from` if it belongs to this Drive instance, otherwise `null`. */
  own(from: DriveEntry): Entry | null;
  meta(ctx: Context): DriveMeta;
  get(ctx: Context, path: string): Entry;
  save?(
    ctx: TaskCtx,
    path: string,
    size: number,
    override: boolean,
    reader: Reader
  ): Entry;
  makeDir?(ctx: Context, path: string): Entry;
  copy?(ctx: TaskCtx, from: Entry, to: string, override: boolean): Entry;
  move?(ctx: TaskCtx, from: Entry, to: string, override: boolean): Entry;
  list(ctx: Context, path: string): Entry[];
  delete?(ctx: TaskCtx, path: string): void;
  upload?(
    ctx: Context,
    path: string,
    size: number,
    override: boolean,
    config: SM
  ): DriveUploadConfig | undefined;

  getReader(
    ctx: Context,
    entry: Entry,
    start: number,
    size: number
  ): ReadCloser;
  getURL?(ctx: Context, entry: Entry): ContentURL;
  hasThumbnail?(entry: Entry): boolean;
  getThumbnail?(ctx: Context, entry: Entry): ReadCloser | ContentURL;
}

declare interface DriveDataStore {
  Save(data: SM): void;
  Load<K extends string, T extends { [key in K]: string | undefined }>(
    key: K,
    ...keys: K[]
  ): T;
}

declare interface DriveCacheItem {
  ModTime: number;
  Size: number;
  Path: string;
  Type: EntryType;
  Data?: SM;
}

declare interface DriveCache {
  PutEntries(entries: Entry[], ttl: Duration): void;
  PutEntry(entry: Entry, ttl: Duration): void;
  PutChildren(parentPath: string, entries: Entry[], ttl: Duration): void;
  Evict(path: string, descendants: boolean): void;
  EvictAll(): void;
  GetEntry(path: string): DriveCacheItem | null;
  GetChildren(path: string): DriveCacheItem[] | null;
}

declare interface DriveInitConfiguration {
  Configured: boolean;
  OAuth?: OAuthConfig;
  Form?: FormItem[];
  Value?: SM;
}

declare interface OAuthConfig {
  URL: string;
  Text: string;
  Principal: string;
}

declare enum OAuthStyle {
  AutoDetect = 0,
  InParams = 1,
  InHeader = 2,
}

declare interface OAuthEndpoint {
  AuthURL: string;
  TokenURL: string;
  AuthStyle?: OAuthStyle;
}

declare interface OAuthRequest {
  Endpoint: OAuthEndpoint;
  RedirectURL: string;
  Scopes: string[];
  Text: string;
}

declare interface OAuthCredentials {
  ClientID: string;
  ClientSecret: string;
}

declare interface OAuthToken {
  AccessToken: string;
  TokenType: string;
  RefreshToken?: string;
  Expiry: GoTime;
}

declare interface OAuthHolder {
  Token(ctx: Context): OAuthToken;
}

declare interface OAuthInitConfigResult {
  Config?: DriveInitConfiguration;
  OAuthHolder?: OAuthHolder;
}

declare interface RootConfig {
  OAuthRedirectURI: string;
  Version: string;
  RevHash: string;
  BuildAt: string;
}

declare interface DriveUtils {
  Config: RootConfig;
  Data: DriveDataStore;
  CreateCache(): DriveCache;
  OAuthInitConfig(
    req: OAuthRequest,
    cred: OAuthCredentials
  ): OAuthInitConfigResult;
  OAuthInit(
    ctx: Context,
    data: SM,
    req: OAuthRequest,
    cred: OAuthCredentials
  ): OAuthHolder | null;
  OAuthLoad(req: OAuthRequest, cred: OAuthCredentials): OAuthHolder;
}

/**
 * Cross-VM shared fields on the instance. Names must start with `$`.
 * Values must be JSON-serializable. Nested mutation is not persisted;
 * reassign the complete property (see `__setData` / `__getData`).
 */
declare type DriveSharedState = {
  [key: `$${string}`]: JSONValue | undefined;
};

declare interface DriveInstanceState extends DriveSharedState {
  /**
   * Raw duration string from the configuration form (e.g. `"2h"`).
   * Parsed by the runtime; empty / invalid / `<= 0` disables entry caching.
   */
  entryCacheTTL?: string;
  /** `meta().Writable`. Defaults to `true` when omitted. */
  writable?: boolean;
}

declare type DriveConfigProps<T> = {
  readonly [K in keyof T as K extends `$${string}` ? never : K]: T[K];
};

declare type DriveSharedProps<T> = {
  [K in keyof T as K extends `$${string}` ? K : never]: T[K];
};

/**
 * Runtime `this` in Drive methods: `createInstance` fields, `cache` / `own`,
 * and Drive operations. Non-`$` fields are frozen; `$` fields stay assignable
 * and are synchronized across VMs.
 */
declare type DriveThis<T extends DriveInstanceState = DriveInstanceState> =
  DriveConfigProps<T> & DriveSharedProps<T> & Drive;

declare interface DriveSetup<T extends DriveInstanceState = DriveInstanceState> {
  /** Static admin configuration form. Fields beginning with `_` are reserved. */
  configForm?: FormItem[];
  /** Validate the static configuration before the runtime instance is created. */
  validateConfig?(config: SM): void;
  /** Return the dynamic initialization state, including an optional form. */
  initConfig?(
    ctx: Context,
    config: SM,
    utils: DriveUtils
  ): DriveInitConfiguration | undefined;
  /** Save or otherwise process submitted dynamic initialization data. */
  init?(ctx: Context, data: SM, config: SM, utils: DriveUtils): void;
  /** Build runtime instance state from static config. Load dynamic data as needed. */
  createInstance(ctx: Context, config: SM, utils: DriveUtils): T;
}

declare interface DriveMethods {
  meta?(ctx: Context): DriveMeta;
  get(ctx: Context, path: string): Entry;
  list(ctx: Context, path: string): Entry[];
  save?(
    ctx: TaskCtx,
    path: string,
    size: number,
    override: boolean,
    reader: Reader
  ): void;
  makeDir?(ctx: Context, path: string): void;
  copy?(ctx: TaskCtx, from: Entry, to: string, override: boolean): void;
  move?(ctx: TaskCtx, from: Entry, to: string, override: boolean): void;
  delete?(ctx: TaskCtx, path: string): void;
  upload?(
    ctx: Context,
    path: string,
    size: number,
    override: boolean,
    config: SM
  ): DriveUploadConfig | undefined;
  getReader?(
    ctx: Context,
    entry: Entry,
    start: number,
    size: number
  ): ReadCloser;
  getURL?(ctx: Context, entry: Entry): ContentURL;
  hasThumbnail?(entry: Entry): boolean;
  getThumbnail?(
    ctx: Context,
    entry: Entry
  ): ReadCloser | ContentURL;
}

/** Prevent `methods` from widening `T`; `this` comes from `createInstance`. */
declare type DriveNoInfer<T> = [T][T extends unknown ? 0 : never];

/**
 * Define a script Drive. `setup` runs before the instance exists;
 * `methods` run on the created instance. `this` is inferred from
 * `createInstance`'s return type (`DriveThis<T>`).
 * The Go runtime supplies entry caching (cache hits do not enter the JS VM),
 * write-path eviction, root `get("")`, copy/move ownership checks, and
 * default `meta` / `upload` / `getReader`.
 */
declare function defineDrive<T extends DriveInstanceState>(
  setup: DriveSetup<T>,
  methods: DriveMethods & ThisType<DriveThis<DriveNoInfer<T>>>
): void;

/** Standard `cache_ttl` form item. Include it in `configForm` to let users set the TTL. */
declare function entryCacheTTLFormItem(defaultValue?: string): FormItem;

declare function useLocalProvider(size: number): DriveUploadConfig;
/**
 * Direct browser upload using this Drive's installed uploader.
 * Do not pass an uploader name; it is taken from the Drive script.
 */
declare function useCustomProvider(
  config?: Record<string, string>
): DriveUploadConfig;

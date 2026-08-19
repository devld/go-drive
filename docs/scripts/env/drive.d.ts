/// <reference path="../global.d.ts"/>

/** This Drive as a host Drive API (for calling back into go-drive). */
declare const selfDrive: DriveInstance;

declare type JSONPrimitive = string | number | boolean | null;
declare type JSONValue =
  | JSONPrimitive
  | JSONValue[]
  | { [key: string]: JSONValue };

/** Drive-level metadata (`meta()`). */
declare interface DriveMeta {
  Writable: boolean;
  Props?: M;
}

/** Entry exchanged between JavaScript and Go. */
declare interface Entry {
  Meta?: EntryMeta;
  IsDir: boolean;
  Path: string;
  Size: number;
  /** Unix milliseconds; use `-1` if unknown. */
  ModTime: number;
  /** Opaque data stored with the cache entry. */
  Data?: SM;
}

declare interface DriveUploadConfig {
  /** `local`, `localChunk`, or `custom`. */
  Provider: string;
  Path?: string;
  Config?: SM;
}

/** Runtime Drive API on `this` after `defineDrive`. */
declare interface Drive {
  cache: DriveCache;
  /** `from` if it belongs to this instance; otherwise `null`. */
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

  /** Range read. Negative `start`/`size` means the whole file. */
  getReader(
    ctx: Context,
    entry: Entry,
    start: number,
    size: number
  ): ReadCloser;
  getURL?(ctx: Context, entry: Entry): ContentURL;
  getThumbnail?(ctx: Context, entry: Entry): ReadCloser | ContentURL;
}

/** Persistent Drive data (tokens, etc.). Keys starting with `_` are reserved. */
declare interface DriveDataStore extends GoHandle<"DriveDataStore"> {
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

declare interface DriveCache extends GoHandle<"DriveCache"> {
  PutEntries(entries: Entry[], ttl: Duration): void;
  PutEntry(entry: Entry, ttl: Duration): void;
  PutChildren(parentPath: string, entries: Entry[], ttl: Duration): void;
  Evict(path: string, descendants: boolean): void;
  EvictAll(): void;
  GetEntry(path: string): DriveCacheItem | null;
  GetChildren(path: string): DriveCacheItem[] | null;
}

/** Dynamic init UI returned by `initConfig`. */
declare interface DriveInitConfiguration {
  Configured: boolean;
  OAuth?: OAuthConfig;
  Form?: FormItem[];
  /** Current values for `Form`. */
  Value?: SM;
}

/** OAuth step shown in the admin UI (not the token holder). */
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
  /** Button label in the admin UI. */
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

/** Persisted OAuth token; refreshes on demand. */
declare interface OAuthHolder extends GoHandle<"OAuthHolder"> {
  /** Current token. Pass the method `ctx` (`Context`, `TaskCtx`, or timeout context); refreshes when expired. */
  Token(ctx: Context): OAuthToken;
}

declare interface OAuthInitConfigResult {
  Config: DriveInitConfiguration & { OAuth: OAuthConfig };
  /** Set when a stored token already exists. */
  OAuthHolder?: OAuthHolder;
}

declare interface RootConfig {
  OAuthRedirectURI: string;
  Version: string;
  RevHash: string;
  BuildAt: string;
}

declare interface DriveUtils extends GoHandle<"DriveUtils"> {
  Config: RootConfig;
  Data: DriveDataStore;
  /** `defineDrive` already assigns `this.cache`. */
  CreateCache(): DriveCache;
  /** Build the OAuth UI step; `OAuthHolder` is set if a token is already stored. */
  OAuthInitConfig(
    req: OAuthRequest,
    cred: OAuthCredentials
  ): OAuthInitConfigResult;
  /** Exchange the submitted auth code and persist the token. */
  OAuthInit(
    ctx: Context,
    data: SM,
    req: OAuthRequest,
    cred: OAuthCredentials
  ): OAuthHolder | null;
  /** Load the persisted token. Throws if the Drive is not configured. */
  OAuthLoad(req: OAuthRequest, cred: OAuthCredentials): OAuthHolder;
}

/**
 * Cross-VM fields. Names must start with `$`. Values must be JSON-serializable.
 * Nested mutation is not persisted; reassign the whole property.
 */
declare type DriveSharedState = {
  [key: `$${string}`]: JSONValue | undefined;
};

declare interface DriveInstanceState extends DriveSharedState {
  /** Form duration (e.g. `"2h"`). Empty / invalid / `<= 0` disables entry cache. */
  entryCacheTTL?: string;
  /** `meta().Writable`. Defaults to `true`. */
  writable?: boolean;
}

declare type DriveConfigProps<T> = {
  readonly [K in keyof T as K extends `$${string}` ? never : K]: T[K];
};

declare type DriveSharedProps<T> = {
  [K in keyof T as K extends `$${string}` ? K : never]: T[K];
};

/**
 * `this` in Drive methods: `createInstance` fields plus `cache` / `own` / ops.
 * Non-`$` fields are frozen; `$` fields stay writable and sync across VMs.
 */
declare type DriveThis<T extends DriveInstanceState = DriveInstanceState> =
  DriveConfigProps<T> & DriveSharedProps<T> & Drive;

declare interface DriveSetup<T extends DriveInstanceState = DriveInstanceState> {
  /** Static admin form. Field names must not start with `_`. */
  configForm?: FormItem[];
  /** Validate static config before `createInstance`. */
  validateConfig?(config: SM): void;
  /** Dynamic init UI (OAuth / extra form). */
  initConfig?(
    ctx: Context,
    config: SM,
    utils: DriveUtils
  ): DriveInitConfiguration | undefined;
  /** Persist submitted dynamic init data. */
  init?(ctx: Context, data: SM, config: SM, utils: DriveUtils): void;
  /** Build instance state from static config. Load dynamic data here. */
  createInstance(ctx: Context, config: SM, utils: DriveUtils): T;
}

/**
 * User-implemented ops. Write methods return void; the runtime re-stats the path.
 * `get` and `list` are required; also implement `getURL` or `getReader`.
 */
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
  getThumbnail?(
    ctx: Context,
    entry: Entry
  ): ReadCloser | ContentURL;
}

/** Keep `this` inferred from `createInstance`, not widened by `methods`. */
declare type DriveNoInfer<T> = [T][T extends unknown ? 0 : never];

/**
 * Define a script Drive. `setup` runs before the instance exists; `methods` run on it.
 * Runtime supplies entry cache, write-path eviction, root `get("")`, copy/move `own` checks,
 * and default `meta` / `upload` / `getReader`.
 */
declare function defineDrive<T extends DriveInstanceState>(
  setup: DriveSetup<T>,
  methods: DriveMethods & ThisType<DriveThis<DriveNoInfer<T>>>
): void;

/** Standard `cache_ttl` form item. */
declare function entryCacheTTLFormItem(defaultValue?: string): FormItem;

/** Server-side upload: `local` or `localChunk` depending on `size`. */
declare function useLocalProvider(size: number): DriveUploadConfig;
/** Direct browser upload via this Drive's installed uploader. Do not pass an uploader name. */
declare function useCustomProvider(
  config?: Record<string, string>
): DriveUploadConfig;

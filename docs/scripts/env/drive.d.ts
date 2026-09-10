/// <reference path="../global.d.ts"/>

/** This script Drive as a host `Drive` (call back into go-drive). Distinct from `this` (`DriveThis`). */
declare const selfDrive: Drive;

declare type JSONValue =
  | JSONPrimitive
  | JSONValue[]
  | { [key: string]: JSONValue };

/** Drive-level metadata (`meta()`). */
declare interface DriveMeta {
  readonly writable: boolean;
  readonly props?: M;
}

/**
 * Plain entry returned by `defineDrive` methods (fields, not the host `Entry` class).
 */
declare interface EntryRecord {
  meta?: EntryMeta;
  isDir: boolean;
  path: string;
  size: number;
  /** Unix milliseconds; use `-1` if unknown. */
  modTime: number;
  /** Opaque data stored with the cache entry. */
  data?: SM;
}

declare interface DriveUploadConfig {
  /** `local`, `localChunk`, or `custom`. */
  provider: string;
  path?: string;
  config?: SM;
}

/** Persistent Drive data (tokens, etc.). Keys starting with `_` are reserved. */
declare interface DriveDataStore extends GoHandle<"DriveDataStore"> {
  save(data: SM): void;
  load<K extends string, T extends { [key in K]: string | undefined }>(
    key: K,
    ...keys: K[]
  ): T;
}

declare interface DriveCacheItem {
  readonly modTime: number;
  readonly size: number;
  readonly path: string;
  readonly type: EntryType;
  readonly data?: SM;
  readonly meta?: EntryMeta;
}

declare interface DriveCache extends GoHandle<"DriveCache"> {
  putEntries(entries: EntryRecord[], ttl: DurationLike): void;
  putEntry(entry: EntryRecord, ttl: DurationLike): void;
  putChildren(
    parentPath: string,
    entries: EntryRecord[],
    ttl: DurationLike
  ): void;
  evict(path: string, descendants: boolean): void;
  evictAll(): void;
  getEntry(path: string): DriveCacheItem | null;
  getChildren(path: string): readonly DriveCacheItem[] | null;
}

/** Dynamic init UI returned by `initConfig`. */
declare interface DriveInitConfiguration {
  readonly configured: boolean;
  readonly oauth?: OAuthConfig;
  readonly form?: readonly FormItem[];
  /** Current values for `form`. */
  readonly value?: SM;
}

/** OAuth step shown in the admin UI (not the token holder). */
declare interface OAuthConfig {
  readonly url: string;
  readonly text: string;
  readonly principal: string;
}

declare enum OAuthStyle {
  AutoDetect = 0,
  InParams = 1,
  InHeader = 2,
}

declare interface OAuthEndpoint {
  authUrl: string;
  tokenUrl: string;
  authStyle?: OAuthStyle;
}

declare interface OAuthRequest {
  endpoint: OAuthEndpoint;
  redirectUrl: string;
  scopes: string[];
  /** Button label in the admin UI. */
  text: string;
}

declare interface OAuthCredentials {
  clientID: string;
  clientSecret: string;
}

declare interface OAuthToken {
  readonly accessToken: string;
  readonly tokenType: string;
  readonly refreshToken?: string;
  readonly expiry: Date;
}

/** Persisted OAuth token; refreshes on demand. */
declare interface OAuthHolder extends GoHandle<"OAuthHolder"> {
  /** Current token; refreshes when expired using the VM run context. */
  token(): OAuthToken;
  /**
   * Force a refresh-token exchange even if the access token is still valid.
   * Updates this holder's memory cache and the Drive data store. Use from
   * `onInterval`; ordinary requests should keep calling `token`.
   */
  refresh(): OAuthToken;
}

declare interface OAuthInitConfigResult {
  readonly config: DriveInitConfiguration & { readonly oauth: OAuthConfig };
  /** Set when a stored token already exists. */
  readonly oauthHolder?: OAuthHolder;
}

declare interface RootConfig {
  readonly oauthRedirectURI: string;
  readonly version: string;
  readonly revHash: string;
  readonly buildAt: string;
}

declare interface DriveUtils extends GoHandle<"DriveUtils"> {
  readonly config: RootConfig;
  data: DriveDataStore;
  /** `defineDrive` already assigns `this.cache`. */
  createCache(): DriveCache;
  /** Build the OAuth UI step; `oauthHolder` is set if a token is already stored. */
  oauthInitConfig(
    req: OAuthRequest,
    cred: OAuthCredentials
  ): OAuthInitConfigResult;
  /** Exchange the submitted auth code and persist the token. */
  oauthInit(
    data: SM,
    req: OAuthRequest,
    cred: OAuthCredentials
  ): OAuthHolder | null;
  /** Load the persisted token. Throws if the Drive is not configured. */
  oauthLoad(req: OAuthRequest, cred: OAuthCredentials): OAuthHolder;
}

/**
 * Cross-VM fields. Names must start with `$`. Values must be JSON-serializable.
 * Values read from Go are read-only views. Nested mutation is not persisted;
 * reassign the whole property.
 */
declare type DriveSharedState = {
  [key: `$${string}`]: JSONValue | undefined;
};

declare interface DriveInterval {
  name: string;
  /** Repeat period. Must be `>= 1ms`. */
  interval: DurationLike;
  /** Per-tick timeout. Defaults to `"30s"`. */
  timeout?: DurationLike;
  /** Run once as soon as the Drive is created. Defaults to `false`. */
  immediately?: boolean;
}

declare interface DriveAdapterState extends DriveSharedState {
  /** Omit, `""`, `null`/`undefined`, or `<= 0` to disable. */
  entryCacheTTL?: DurationLike;
  /** `meta().writable`. Defaults to `true`. */
  writable?: boolean;
  /**
   * Repeating background work started with the Drive instance and stopped on dispose.
   * Requires `onInterval`. Do not use Admin Jobs for Drive-internal keep-alive.
   */
  intervals?: DriveInterval[];
}

declare type DriveConfigProps<T> = {
  readonly [K in keyof T as K extends `$${string}` ? never : K]: T[K];
};

declare type DriveSharedProps<T> = {
  [K in keyof T as K extends `$${string}` ? K : never]: T[K];
};

/**
 * User-implemented ops. Write methods return void; the runtime re-stats the path.
 * `get` and `list` are required; also implement `getURL` or `getReader`.
 */
declare interface DriveMethods {
  meta?(): DriveMeta;
  get(path: string): EntryRecord;
  list(path: string): EntryRecord[];
  save?(
    path: string,
    size: number,
    override: boolean,
    reader: Reader,
    progress: ProgressReporter
  ): void;
  makeDir?(path: string): void;
  copy?(
    from: EntryRecord,
    to: string,
    override: boolean,
    progress: ProgressReporter
  ): void;
  move?(
    from: EntryRecord,
    to: string,
    override: boolean,
    progress: ProgressReporter
  ): void;
  delete?(path: string, progress: ProgressReporter): void;
  upload?(
    path: string,
    size: number,
    override: boolean,
    config: SM
  ): DriveUploadConfig | undefined;
  getReader?(
    entry: EntryRecord,
    start: number,
    size: number
  ): Reader;
  getURL?(entry: EntryRecord): ContentURL;
  getThumbnail?(entry: EntryRecord): Reader | ContentURL;
  /**
   * Periodic work declared in `createInstance.intervals`. Go owns the clock and
   * borrows a VM like other methods. Return `"25m"` or `ms(...)` to reschedule;
   * omit to keep `interval`. Keep this short. For OAuth, call
   * `this.oauth.refresh()` on the holder from `createInstance` — do not
   * `oauthLoad` again.
   */
  onInterval?(name: string): DurationLike | void;
}

/**
 * `this` in adapter methods: `createInstance` fields, injected `cache`,
 * and bound ops. Distinct from the host `Drive` class (`selfDrive`).
 * Non-`$` fields are frozen; `$` fields stay writable and sync across VMs.
 * `cache` is runtime-injected; do not pass it in `DriveMethods`.
 */
declare type DriveThis<T extends DriveAdapterState = DriveAdapterState> =
  DriveConfigProps<T> &
    DriveSharedProps<T> &
    DriveMethods & {
      readonly cache: DriveCache;
    };

declare interface DriveSetup<T extends DriveAdapterState = DriveAdapterState> {
  /** Static admin form. Field names must not start with `_`. */
  configForm?: FormItem[];
  /** Validate static config before `createInstance`. */
  validateConfig?(config: SM): void;
  /** Dynamic init UI (OAuth / extra form). */
  initConfig?(
    config: SM,
    utils: DriveUtils
  ): DriveInitConfiguration | undefined;
  /** Persist submitted dynamic init data. */
  init?(data: SM, config: SM, utils: DriveUtils): void;
  /** Build instance state from static config. Load dynamic data here. */
  createInstance(config: SM, utils: DriveUtils): T;
}

/** Keep `this` inferred from `createInstance`, not widened by `methods`. */
declare type DriveNoInfer<T> = [T][T extends unknown ? 0 : never];

/**
 * Define a script Drive. `setup` runs before the instance exists; `methods` run on it.
 * Runtime supplies entry cache, write-path eviction, root `get("")`, copy/move ownership checks,
 * and default `meta` / `upload` / `getReader`.
 */
declare function defineDrive<T extends DriveAdapterState>(
  setup: DriveSetup<T>,
  methods: DriveMethods & ThisType<DriveThis<DriveNoInfer<T>>>
): void;

/** Standard `cache_ttl` form item. */
declare function entryCacheTTLFormItem(defaultValue?: string): FormItem;

/** 5 MiB. `useLocalProvider` uses `localChunk` above this size. */
declare const LOCAL_PROVIDER_CHUNK_SIZE: number;

/** Server-side upload: `local` or `localChunk` depending on `size`. */
declare function useLocalProvider(size: number): DriveUploadConfig;
/** Direct browser upload via this Drive's installed uploader. Do not pass an uploader name. */
declare function useCustomProvider(
  config?: Record<string, string>
): DriveUploadConfig;

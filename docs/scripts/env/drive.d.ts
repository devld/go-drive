/// <reference path="../global.d.ts"/>

declare const selfDrive: DriveInstance;

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
  copy?(ctx: TaskCtx, from: DriveEntry, to: string, override: boolean): Entry;
  move?(ctx: TaskCtx, from: DriveEntry, to: string, override: boolean): Entry;
  list(ctx: Context, path: string): Entry[];
  delete?(ctx: TaskCtx, path: string): void;
  upload?(
    ctx: Context,
    path: string,
    size: number,
    override: boolean,
    config: SM
  ): DriveUploadConfig | undefined;

  getReader(ctx: Context, entry: Entry, start: number, size: number): ReadCloser;
  getURL?(ctx: Context, entry: Entry): ContentURL;
  hasThumbnail?(entry: Entry): boolean;
  getThumbnail?(ctx: Context, entry: Entry): ReadCloser | ContentURL;
}

declare interface DriveDataStore {
  Save(data: SM): void;
  Load<K extends string, T extends { [key in K]: string | undefined }>(...keys: K[]): T;
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
  Form: FormItem[];
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

declare interface OAuthResponse {
  Token(): OAuthToken;
}

declare interface OAuthInitConfigResult {
  Config?: DriveInitConfiguration;
  Response?: OAuthResponse;
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
  ): OAuthResponse | null;
  OAuthGet(req: OAuthRequest, cred: OAuthCredentials): OAuthResponse;
}

declare type DriveCreate = (
  ctx: Context,
  config: SM,
  utils: DriveUtils
) => Drive;

declare type DriveInitConfig = (
  ctx: Context,
  config: SM,
  utils: DriveUtils
) => DriveInitConfiguration;

declare type DriveInit = (
  ctx: Context,
  data: SM,
  config: SM,
  utils: DriveUtils
) => void;

/** Define the function used to create the Drive instance. This is required. */
declare function defineCreate(fn: DriveCreate): void;
/** Define the function used to get the initialization data of this Drive. This is optional. */
declare function defineInitConfig(fn: DriveInitConfig): void;
/** Define the function used to initialize this Drive. This is optional if `defineInitConfig` is not present. */
declare function defineInit(fn: DriveInit): void;

declare function useLocalProvider(size: number): DriveUploadConfig;
declare function useCustomProvider(
  uploader: string,
  config?: Record<string, string>
): DriveUploadConfig;

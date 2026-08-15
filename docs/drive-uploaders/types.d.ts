declare interface HttpUploadProgress {
  loaded: number;
  total: number;
}

declare type HttpRequestMethod =
  | "get"
  | "delete"
  | "head"
  | "options"
  | "post"
  | "put"
  | "patch";

export type HttpRequestTransformer = (
  config: HttpRequestConfig
) => Promise<HttpRequestConfig> | HttpRequestConfig;
export type HttpResponseTransformer<DT = any> = (
  error: any,
  resp: HttpResponse
) => DT | Promise<DT>;

declare interface HttpRequestConfig {
  url?: string;
  method?: HttpRequestMethod;
  data?: any;
  headers?: Record<string, any>;
  timeout?: number;
  transformRequest?: HttpRequestTransformer | HttpRequestTransformer[];
  transformResponse?: HttpResponseTransformer | HttpResponseTransformer[];
  params?: any;

  onUploadProgress?: (p: HttpUploadProgress) => void;
}

declare interface HttpResponse<DT = any> {
  status: number;
  headers: Record<string, any>;
  data?: DT;

  request: HttpRequestConfig;
}

declare interface Http<T = any> {
  <DT = T>(config: HttpRequestConfig): Promise<DT>;

  head: <DT = T>(url: string, config?: HttpRequestConfig) => Promise<DT>;
  get: <DT = T>(url: string, config?: HttpRequestConfig) => Promise<DT>;
  post: <DT = T>(
    url: string,
    data?: any,
    config?: HttpRequestConfig
  ) => Promise<DT>;
  put: <DT = T>(
    url: string,
    data?: any,
    config?: HttpRequestConfig
  ) => Promise<DT>;
  delete: <DT = T>(url: string, config?: HttpRequestConfig) => Promise<DT>;
}

declare interface CustomUploader {
  /** Returns the number of chunks to upload. Omit to upload the file as a single request. */
  prepare?(): Promise<number>;
  /** Returns the blob for chunk `seq` (0-based). Omit to use the whole file. */
  getChunk?(seq: number): Blob;
  /** Uploads one chunk. `onProgress` reports this chunk's uploaded bytes. */
  upload(
    data: Blob,
    seq: number,
    onProgress: (p: HttpUploadProgress) => void
  ): Promise<any>;
  /** Called after every chunk succeeds. Use it to commit a multipart upload. */
  complete?(): Promise<any>;
  /** Called after success, cancel, or prepare/upload failure. Abort remote multipart state here. */
  onCleanup?(): void;
}

declare interface TaskDef {
  /** Destination path on the Drive. */
  path: string;
  /** File being uploaded. */
  file?: Blob;
  /** File size in bytes. */
  size?: number;
  /** Whether an existing file at `path` may be overwritten. */
  override?: boolean;
}

declare type UploadCallback = <T = any>(
  data: Record<string, string>
) => Promise<T>;

declare interface UploadFactoryContext {
  /**
   * Values from the Drive script's `upload()` / `useCustomProvider` config.
   * Typically short-lived upload credentials and target URLs, not account secrets.
   */
  readonly config: Record<string, string>;
  /**
   * HTTP helper used for requests to the storage service.
   * Requests are tracked by the upload task so pause/cancel can abort them.
   * Pass `http` as the second argument to use go-drive's authenticated API client instead.
   */
  readonly request: <T>(config: HttpRequestConfig, http?: Http) => Promise<T>;
  /**
   * Maximum number of chunks uploaded in parallel. Writable; values are clamped.
   */
  maxConcurrent: number;
  /**
   * Authenticated HTTP client for the go-drive API (same origin, current user).
   * Prefer `request` for storage uploads and `uploadCallback` for Drive notifications.
   */
  readonly http: Http;
  /** The current upload task (path, file blob, size, override). */
  readonly task: TaskDef;
  /**
   * Calls the Drive script's `upload()` again (POST /prepare-upload) with extra data,
   * e.g. `{ action: "Completed" }`, so the server can confirm the object and evict caches.
   */
  readonly uploadCallback: UploadCallback;
}

declare type UploadFactory = (ctx: UploadFactoryContext) => CustomUploader;

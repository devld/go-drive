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
   * HTTP helper for the storage service. Requests are tracked so pause/cancel abort them.
   */
  readonly request: <T>(config: HttpRequestConfig) => Promise<T>;
  /**
   * Maximum number of chunks uploaded in parallel. Writable; values are clamped.
   */
  maxConcurrent: number;
  /** The current upload task (path, file blob, size, override). */
  readonly task: TaskDef;
  /** Chunk count computed from `chunkSize` and `task.file.size`. */
  chunks: number;
  /**
   * Calls the Drive script's `upload()` again (POST /prepare-upload).
   * After a successful `complete`, the runtime already sends `{ action: "Completed" }`.
   */
  readonly uploadCallback: UploadCallback;
}

declare interface UploaderUploadArgs {
  blob: Blob;
  /** 0-based chunk index. */
  seq: number;
  /** Value returned by `start`, or `null`/`undefined` for a single-request upload. */
  session: any;
  onProgress: (p: HttpUploadProgress) => void;
}

declare interface UploaderCompleteArgs {
  session: any;
  /** `upload` return values, indexed by `seq`. */
  parts: any[];
}

declare interface UploaderAbortArgs {
  session: any;
}

declare interface UploaderSpec {
  /** When set, the runtime slices the file and sets `ctx.chunks`. Omit to upload in one request. */
  chunkSize?: number;
  maxConcurrent?: number;
  /** Create remote multipart state. Return session data, or `null` for a simple upload. */
  start?(ctx: UploadFactoryContext): Promise<any>;
  upload(ctx: UploadFactoryContext, args: UploaderUploadArgs): Promise<any>;
  /** Commit a multipart upload. The runtime then notifies the Drive with `action: "Completed"`. */
  complete?(ctx: UploadFactoryContext, args: UploaderCompleteArgs): Promise<any>;
  /** Abort remote multipart state after cancel or failure. Not called after a successful complete. */
  abort?(ctx: UploadFactoryContext, args: UploaderAbortArgs): void | Promise<void>;
  /** Override default `file.slice`. Rarely needed. */
  getChunk?(ctx: UploadFactoryContext, seq: number): Blob;
}

declare function defineUploader(spec: UploaderSpec): void;

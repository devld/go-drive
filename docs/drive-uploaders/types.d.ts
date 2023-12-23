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
  prepare?(): Promise<number>;
  getChunk?(seq: number): Blob;
  upload(
    data: Blob,
    seq: number,
    onProgress: (p: HttpUploadProgress) => void
  ): Promise<any>;
  complete?(): Promise<any>;

  onCleanup?(): void;
}

declare interface TaskDef {
  path: string;
  file?: Blob;
  size?: number;
  override?: boolean;
}

declare type UploadCallback = <T = any>(
  data: Record<string, string>
) => Promise<T>;

declare interface UploadFactoryContext {
  readonly config: Record<string, string>;
  readonly request: <T>(config: HttpRequestConfig, http?: Http) => Promise<T>;
  maxConcurrent: number;
  readonly http: Http;
  readonly task: TaskDef;
  readonly uploadCallback: UploadCallback;
}

declare type UploadFactory = (ctx: UploadFactoryContext) => CustomUploader;

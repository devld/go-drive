declare interface UploadProgress {
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

declare type HttpDataTransformer = (
  data: any,
  headers?: Record<string, string>
) => any;

declare interface HttpRequestConfig {
  url?: string;
  method?: HttpRequestMethod;
  data?: any;
  headers?: Record<string, any>;
  timeout?: number;
  transformRequest?: HttpDataTransformer | HttpDataTransformer[];
  transformResponse?: HttpDataTransformer | HttpDataTransformer[];
  params?: any;

  onUploadProgress?: (p: UploadProgress) => void;
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
    onProgress: (p: UploadProgress) => void
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

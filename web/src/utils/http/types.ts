export type HttpRequestMethod =
  | 'get'
  | 'delete'
  | 'head'
  | 'options'
  | 'post'
  | 'put'
  | 'patch'

export interface HttpUploadProgress {
  loaded: number
  total: number
}

export type HttpRequestTransformer = (
  config: HttpRequestConfig
) => Promise<HttpRequestConfig> | HttpRequestConfig
export type HttpResponseTransformer<DT = any> = (
  error: any,
  resp: HttpResponse
) => DT | Promise<DT>

export interface HttpRequestBaseConfig {
  baseURL?: string
  headers?: Record<string, any>
  timeout?: number
  params?: any
  transformRequest?: HttpRequestTransformer | HttpRequestTransformer[]
  transformResponse?: HttpResponseTransformer | HttpResponseTransformer[]

  context?: Record<string, any>
}

export interface HttpRequestConfig extends HttpRequestBaseConfig {
  url?: string
  method?: HttpRequestMethod
  data?: any

  onUploadProgress?: (p: HttpUploadProgress) => void
}

export interface HttpResponse<DT = any> {
  status: number
  headers: Record<string, any>
  data?: DT

  request: HttpRequestConfig
}

export interface RequestTask<T = HttpResponse> {
  get promise(): Promise<T> | undefined

  then<TResult1 = T, TResult2 = never>(
    onfulfilled?:
      | ((value: T) => TResult1 | PromiseLike<TResult1>)
      | undefined
      | null,
    onrejected?:
      | ((reason: any) => TResult2 | PromiseLike<TResult2>)
      | undefined
      | null
  ): RequestTask<TResult1 | TResult2> & Promise<TResult1 | TResult2>

  catch<TResult = never>(
    onrejected?:
      | ((reason: any) => TResult | PromiseLike<TResult>)
      | undefined
      | null
  ): RequestTask<T | TResult> & Promise<T | TResult>

  finally(
    onfinally?: (() => void) | undefined | null
  ): RequestTask<T> & Promise<T>

  cancel(message?: string): void

  get [Symbol.toStringTag](): string
}

export interface HttpBase<T = any> {
  <DT = T>(config: HttpRequestConfig): RequestTask<DT>
}

export interface HttpExtend<T = any> {
  head: <DT = T>(url: string, config?: HttpRequestConfig) => RequestTask<DT>
  get: <DT = T>(url: string, config?: HttpRequestConfig) => RequestTask<DT>
  post: <DT = T>(
    url: string,
    data?: any,
    config?: HttpRequestConfig
  ) => RequestTask<DT>
  put: <DT = T>(
    url: string,
    data?: any,
    config?: HttpRequestConfig
  ) => RequestTask<DT>
  delete: <DT = T>(url: string, config?: HttpRequestConfig) => RequestTask<DT>
}

export type Http<T = any> = HttpBase<T> & HttpExtend<T>

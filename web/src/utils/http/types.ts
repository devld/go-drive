import type { RequestTask } from './utils'

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

export type HttpDataTransformer = (data: any, headers?: O) => any

export interface HttpRequestConfig {
  url?: string
  method?: HttpRequestMethod
  data?: any
  headers?: O
  timeout?: number
  transformRequest?: HttpDataTransformer | HttpDataTransformer[]
  transformResponse?: HttpDataTransformer | HttpDataTransformer[]
  params?: any

  onUploadProgress?: (p: HttpUploadProgress) => void
}

export interface Http<T = any> {
  <DT = T>(config: HttpRequestConfig): RequestTask<DT>

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

import Axios, {
  AxiosError,
  AxiosInstance,
  CancelTokenSource,
  InternalAxiosRequestConfig,
} from 'axios'
import { Http, HttpRequestConfig } from './types'

export class ApiError extends Error {
  static from(e: AxiosError<any>) {
    if (!e.response) return e
    const status = e.response.status
    const res = e.response.data
    return new ApiError(status, res.message, res.data)
  }

  constructor(
    readonly status: number,
    readonly message: string,
    readonly data?: any,
    readonly _isCancel?: boolean
  ) {
    super(message)
  }

  get isCancel() {
    return this._isCancel
  }
}

export class RequestTask<T = any> {
  static from<T>(v: PromiseValue<T>, source: CancelTokenSource) {
    const t = new RequestTask<T>(source)
    t._setPromise(v)
    return t
  }

  static wrap<T>(fn: (t: RequestTask<T>) => PromiseValue<T>) {
    const task = new RequestTask<T>()
    task._setPromise(fn(task))
    return task
  }

  private _promise?: PromiseValue<T>
  private _axiosSource: CancelTokenSource

  constructor(axiosSource?: CancelTokenSource) {
    this._promise = undefined
    this._axiosSource = axiosSource || Axios.CancelToken.source()
    this.then = this.then.bind(this)
    this.catch = this.catch.bind(this)
    this.finally = this.finally.bind(this)
  }

  _setPromise(v: PromiseValue<T>) {
    if (this._promise) throw new Error('_setPromise already called')
    this._promise = Promise.resolve(v)
  }

  get promise() {
    return this._promise
  }

  get token() {
    return this._axiosSource.token
  }

  then<TResult1 = T, TResult2 = never>(
    onfulfilled?:
      | ((value: T) => TResult1 | PromiseLike<TResult1>)
      | undefined
      | null,
    onrejected?:
      | ((reason: any) => TResult2 | PromiseLike<TResult2>)
      | undefined
      | null
  ): RequestTask<TResult1 | TResult2> & Promise<TResult1 | TResult2> {
    return RequestTask.from<TResult1 | TResult2>(
      Promise.resolve(this._promise!).then(onfulfilled, onrejected),
      this._axiosSource
    )
  }

  catch<TResult = never>(
    onrejected?:
      | ((reason: any) => TResult | PromiseLike<TResult>)
      | undefined
      | null
  ): RequestTask<T | TResult> & Promise<T | TResult> {
    return RequestTask.from<T | TResult>(
      Promise.resolve(this._promise!).catch(onrejected),
      this._axiosSource
    )
  }

  finally(
    onfinally?: (() => void) | undefined | null
  ): RequestTask<T> & Promise<T> {
    return RequestTask.from<T>(
      Promise.resolve(this._promise!).finally(onfinally),
      this._axiosSource
    )
  }

  cancel(message?: string) {
    this._axiosSource.cancel(message)
  }

  get [Symbol.toStringTag]() {
    return Promise.resolve(this._promise)[Symbol.toStringTag] ?? ''
  }
}

function wrapConfig<T>(
  task: RequestTask<T>,
  config: HttpRequestConfig | undefined,
  axios: any
) {
  const config_ = { ...config } as InternalAxiosRequestConfig

  config_.cancelToken = task.token
  config_._axios = axios
  return config_
}

export function wrapAxios<T = any>(axios: AxiosInstance): Http<T> {
  const wrapper = function <DT = T>(config: HttpRequestConfig) {
    return RequestTask.wrap<DT>((t) =>
      axios.request<any, DT>(wrapConfig(t, config, wrapper))
    )
  }
  wrapper.head = function <DT = T>(url: string, config?: HttpRequestConfig) {
    return RequestTask.wrap<DT>((t) =>
      axios.head(url, wrapConfig(t, config, wrapper))
    )
  }
  wrapper.get = function <DT = T>(url: string, config?: HttpRequestConfig) {
    return RequestTask.wrap<DT>((t) =>
      axios.get(url, wrapConfig(t, config, wrapper))
    )
  }
  wrapper.post = function <DT = T>(
    url: string,
    data?: any,
    config?: HttpRequestConfig
  ) {
    return RequestTask.wrap<DT>((t) =>
      axios.post(url, data, wrapConfig(t, config, wrapper))
    )
  }

  wrapper.put = function <DT = T>(
    url: string,
    data: any,
    config?: HttpRequestConfig
  ) {
    return RequestTask.wrap<DT>((t) =>
      axios.put(url, data, wrapConfig(t, config, wrapper))
    )
  }
  wrapper.delete = function <DT = T>(url: string, config?: HttpRequestConfig) {
    return RequestTask.wrap<DT>((t) =>
      axios.delete(url, wrapConfig(t, config, wrapper))
    )
  }

  return wrapper
}

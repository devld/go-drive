import { RequestTask } from './types'

export class HttpError extends Error {
  constructor(
    readonly status: number,
    readonly message: string,
    readonly response?: any,
    readonly _isCancel?: boolean
  ) {
    super(message)
  }

  get isCancel() {
    return this._isCancel
  }
}

export class RequestTaskImpl<T = any> {
  static from<T>(v: PromiseValue<T>, aborter: AbortController) {
    const t = new RequestTaskImpl<T>(aborter)
    t._setPromise(v)
    return t
  }

  private _promise?: Promise<T>
  private _aborter: AbortController

  constructor(aborter?: AbortController) {
    this._promise = undefined
    this._aborter = aborter || new AbortController()
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
    return RequestTaskImpl.from<TResult1 | TResult2>(
      Promise.resolve(this._promise!).then(onfulfilled, onrejected),
      this._aborter
    )
  }

  catch<TResult = never>(
    onrejected?:
      | ((reason: any) => TResult | PromiseLike<TResult>)
      | undefined
      | null
  ): RequestTask<T | TResult> & Promise<T | TResult> {
    return RequestTaskImpl.from<T | TResult>(
      Promise.resolve(this._promise!).catch(onrejected),
      this._aborter
    )
  }

  finally(
    onfinally?: (() => void) | undefined | null
  ): RequestTask<T> & Promise<T> {
    return RequestTaskImpl.from<T>(
      Promise.resolve(this._promise!).finally(onfinally),
      this._aborter
    )
  }

  cancel(message?: string) {
    this._aborter.abort(message)
  }

  get [Symbol.toStringTag]() {
    return Promise.resolve(this._promise)[Symbol.toStringTag] ?? ''
  }
}

export const normalizeHttpHeader = (headers: Headers) => {
  const result: O = {}
  headers.forEach((value, key) => {
    if (typeof result[key] === 'undefined') {
      result[key] = value
    } else {
      if (!Array.isArray(result[key])) result[key] = [result[key]]
      result[key].push(value)
    }
  })
  return result
}

export const normalizeXHRHttpHeader = (headers: string) => {
  const result: O = {}
  headers
    .split('\r\n')
    .map((line) => line.split(': '))
    .forEach(([key, value]) => {
      if (typeof result[key] === 'undefined') {
        result[key] = value
      } else {
        if (!Array.isArray(result[key])) result[key] = [result[key]]
        result[key].push(value)
      }
    })
  return result
}

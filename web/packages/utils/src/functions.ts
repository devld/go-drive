import type { AnyRecord } from './types'

export function stringSplitN(s: string, delim: string | RegExp, n: number) {
  if (n <= 0) return s.split(s)
  if (typeof delim === 'object') {
    delim = new RegExp(delim, 'g')
  }
  const result: string[] = []
  let i = 0
  let matchedLen = 0
  while (i < s.length) {
    if (result.length === n - 1) break
    let p: number
    if (typeof delim === 'string') {
      p = s.indexOf(delim, i)
      matchedLen = delim.length
    } else {
      const r = delim.exec(s)
      p = r?.index ?? -1
      if (r) matchedLen = r[0].length
    }
    if (p === -1) break
    result.push(s.slice(i, p))
    i = p + matchedLen
  }
  result.push(s.slice(i))
  return result
}

export function encodeQuery(q: AnyRecord) {
  if (!q || typeof q !== 'object') return
  return Object.keys(q)
    .filter((key) => q[key] !== undefined)
    .map(
      (key) =>
        `${encodeURIComponent(key)}=${
          q[key] === null ? '' : encodeURIComponent(q[key]) ?? ''
        }`
    )
    .join('&')
}

export const isPlainObject = (value: unknown) =>
  Object.prototype.toString.call(value) === '[object Object]'

export function buildURL(url: string, q: AnyRecord) {
  if (typeof url !== 'string') return
  const encodedQ = encodeQuery(q) || ''
  const m = /^([^?#]*)(\?([^#]*))?(#(.*))?$/.exec(url)!
  url = m[1] || ''
  let qs = ''
  if (m[3] || encodedQ) {
    qs = m[3] || ''
    if (qs && !qs.endsWith('&') && encodedQ) qs += '&'
    qs += encodedQ
  }
  if (qs) url += '?' + qs
  if (m[5]) url += '#' + (m[5] || '')
  return url
}

export function arrayRemove<T>(array: T[], predicate: (value: T) => boolean) {
  const index = array.findIndex(predicate)
  let value
  if (index >= 0) {
    value = array[index]
    array.splice(index, 1)
  }
  return value
}

export const debounce = <
  Args extends any[],
  F extends (...args: Args) => any,
  This
>(
  func: F,
  wait: number
): ((...args: Args) => void) => {
  let timeout: number | undefined
  return function (this: This, ...params) {
    const later = () => {
      timeout = undefined
      func.apply(this, params)
    }
    clearTimeout(timeout!)
    timeout = setTimeout(later, wait) as unknown as number
  }
}

export interface ThrottleOptions {
  leading?: boolean
  trailing?: boolean
}

export function throttle<
  Args extends any[],
  Result,
  F extends (...args: Args) => any,
  This
>(func: F, wait: number, options?: ThrottleOptions): (...args: Args) => Result {
  let context: This | null
  let args: Args | null
  let result: Result | null
  let timeout: number | undefined
  let previous = 0
  if (!options) options = {}
  const later = function () {
    previous = options?.leading === false ? 0 : Date.now()
    timeout = undefined
    result = func.apply(context, args as Args)
    if (!timeout) {
      context = null
      args = null
    }
  }
  return function (this: This, ...params): Result {
    const now = Date.now()
    if (!previous && options?.leading === false) previous = now
    const remaining = wait - (now - previous)
    args = params
    if (remaining <= 0 || remaining > wait) {
      if (timeout) {
        clearTimeout(timeout)
        timeout = undefined
      }
      previous = now
      result = func.apply(context, args)
      if (!timeout) context = args = null
    } else if (!timeout && options?.trailing !== false) {
      timeout = setTimeout(later, remaining) as unknown as number
    }
    return result as any
  }
}

export function waitPromise(fn: (...args: any[]) => Promise<any>) {
  let promise: Promise<any> | undefined
  return function (this: any, ...params: any[]) {
    if (!promise) {
      promise = fn.apply(this, params).finally(() => {
        promise = undefined
      })
    }
    return promise
  }
}

export const IDENTICAL = (value: any) => value

export function mapOf<T, R = T>(
  list: Readonly<T[]>,
  keyFn: (value: T, index: number, list: Readonly<T[]>) => string,
  valueFn: (
    value: T,
    index: number,
    list: Readonly<T[]>
  ) => R = IDENTICAL
) {
  const map: { [key: string]: R } = {}
  list.forEach((value, index, all) => {
    map[keyFn(value, index, all)] = valueFn(value, index, all)
  })
  return map
}

export function val<T>(value: T | undefined, defaultValue: T) {
  if (value === undefined) return defaultValue
  return value
}

export function wait(ms: number) {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })
}

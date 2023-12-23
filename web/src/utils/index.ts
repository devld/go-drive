import { getTask } from '@/api'

import focus from './directives/focus'
import longPress from './directives/long-press'
import markdown from './directives/markdown'
import lazySrc from './directives/lazy-src'
import { Entry, Task, User } from '../types'
import { Directive, Plugin } from 'vue'
import { LocationQuery } from 'vue-router'

export const IS_DEBUG = process.env.NODE_ENV === 'development'

export function setTitle(title?: I18nText) {
  if (title) title += ' - ' + window.___config___.appName
  else title = window.___config___.appName
  document.title = title.toString()
}

export function formatTime(d: any) {
  if (!d) return ''
  if (!(d instanceof Date)) d = new Date(d)
  if (isNaN(d.getTime())) return ''
  if (d.getTime() < 0) return ''
  const year = d.getFullYear()
  let month = d.getMonth() + 1
  let day = d.getDate()
  let hour = d.getHours()
  let minute = d.getMinutes()
  let second = d.getSeconds()
  month = month < 10 ? '0' + month : month
  day = day < 10 ? '0' + day : day
  hour = hour < 10 ? '0' + hour : hour
  minute = minute < 10 ? '0' + minute : minute
  second = second < 10 ? '0' + second : second
  return `${year}-${month}-${day} ${hour}:${minute}:${second}`
}

// from https://stackoverflow.com/a/18650828/8749466
export function formatBytes(bytes: number, decimals = 2) {
  if (typeof bytes !== 'number') return ''
  if (bytes < 0) return '-'
  if (bytes === 0) return '0 B'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'K', 'M', 'G', 'T']

  let i = Math.floor(Math.log(bytes) / Math.log(k))
  if (i >= sizes.length) i = sizes.length - 1

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

export function formatPercent(n: number, based?: number) {
  if (typeof n !== 'number') return ''
  if (typeof based === 'number') {
    if (based === 0) return ''
    n /= based
  }
  return (n * 100).toFixed(1) + '%'
}

export function stringSplitN(s: string, delim: string | RegExp, n: number) {
  if (n <= 0) return s.split(s)
  if (typeof delim === 'object') {
    delim = new RegExp(delim, 'g')
  }
  const result = []
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
    if (p === -1) {
      break
    }
    result.push(s.slice(i, p))
    i = p + matchedLen
  }
  result.push(s.slice(i))
  return result
}

export function isParentPath(path: string, parent: string) {
  if (isRootPath(path)) return false
  if (isRootPath(parent)) return true
  return path.startsWith(`${parent}/`)
}

export function dir(path: string) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return ''
  return path.substring(0, i)
}

export function filename(path: string) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return path
  return path.substring(i + 1)
}

export function filenameBase(filename?: string) {
  if (!filename) return ''
  const i = filename.lastIndexOf('.')
  if (i === -1) return filename
  return filename.substring(0, i)
}

export function filenameExt(filename?: string) {
  if (!filename) return ''
  const i = filename.lastIndexOf('.')
  if (i === -1) return ''
  return filename.substring(i + 1).toLowerCase()
}

export function pathJoin(...segments: string[]) {
  return segments.filter(Boolean).join('/').replace(/\/+/g, '/')
}

export function pathClean(path: string) {
  if (!path) return ''
  const segments = path.split('/').filter(Boolean)
  const paths: string[] = []
  segments.forEach((s) => {
    if (s === '.') return
    if (s === '..') paths.pop()
    else paths.push(s)
  })
  return paths.join('/')
}

export function entryMatches(
  entry: Entry | string,
  matches: string | readonly string[]
) {
  const name = typeof entry === 'object' ? entry.name : entry
  const meta = typeof entry === 'object' ? entry.meta : undefined
  const matches_ = (Array.isArray(matches) ? matches : [matches]) as string[]
  for (const m of matches_) {
    // matches full filename
    if (m.startsWith('/') && name.toLowerCase() === m.substring(1)) {
      return true
    }
    // multiple extensions
    if (m.includes('.') && name.toLowerCase().endsWith('.' + m)) {
      return true
    }
    // match ext
    const ext = meta?.ext || filenameExt(name)
    if (m === ext) return true
  }
  return false
}

export function createEntryExtMatcher<T extends string = string>(
  extsMap: Record<T, string[]>
): (entry: Entry | string) => T | undefined {
  const extMapping: Record<string, T> = {}
  const fullNameMapping: Record<string, T> = {}
  Object.keys(extsMap).forEach((icon_) => {
    const icon = icon_ as T
    extsMap[icon].forEach((ext) => {
      if (ext.startsWith('/')) {
        fullNameMapping[ext.substring(1)] = icon
      } else {
        extMapping[ext] = icon
      }
    })
  })
  return (entry: Entry | string) => {
    const name = typeof entry === 'object' ? entry.name : entry
    const meta = typeof entry === 'object' ? entry.meta : undefined
    let icon = fullNameMapping[name.toLowerCase()]
    if (!icon) {
      const ext = meta?.ext || filenameExt(name)
      icon = extMapping[ext]
    }
    return icon
  }
}

export function getRouteQuery(q: LocationQuery, key: string) {
  const d = q[key]
  return Array.isArray(d) ? d[0] : d
}

export function encodeQuery(q: O) {
  if (!q || typeof q !== 'object') return
  return Object.keys(q)
    .map((k) => `${encodeURIComponent(k)}=${encodeURIComponent(q[k]) ?? ''}`)
    .join('&')
}

export const isPlainObject = (o: any) =>
  Object.prototype.toString.call(o) === '[object Object]'

export function buildURL(url: string, q: O) {
  if (typeof url !== 'string') return
  const encodedQ = encodeQuery(q) || ''
  const m = /^([^?#]*)(\?([^#]*))?(#(.*))?$/.exec(url)!
  url = m[1] || ''
  let qs = ''
  if (m[3] || encodedQ) {
    qs = m[3] || ''
    if (qs && !qs.endsWith('&') && encodedQ) {
      qs += '&'
    }
    qs += encodedQ
  }
  if (qs) url += '?' + qs
  if (m[5]) {
    url += '#' + (m[5] || '')
  }
  return url
}

export function arrayRemove<T>(array: T[], e: Fn1<T, boolean>) {
  const index = array.findIndex(e)
  let el
  if (index >= 0) {
    el = array[index]
    array.splice(index, 1)
  }
  return el
}

export const debounce = <
  Args extends any[],
  F extends (...args: Args) => any,
  THIS
>(
  func: F,
  wait: number
): ((...args: Args) => void) => {
  let timeout: number | undefined
  return function (this: THIS, ...params) {
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
  THIS
>(func: F, wait: number, options?: ThrottleOptions): (...args: Args) => Result {
  let context: THIS | null, args: Args | null, result: Result | null
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
  return function (this: THIS, ...params): Result {
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

export function waitPromise<T = any>(fn: Fnn) {
  let promise: Promise<T> | undefined
  return function (this: any, ...params: any[]) {
    if (!promise) {
      promise = fn.apply(this, params).finally(() => {
        promise = undefined
      })
    }
    return promise
  }
}

export const IDENTICAL = (e: any) => e

export function mapOf<T, R = T>(
  list: Readonly<T[]>,
  keyFn: (e: T, i: number, a: Readonly<T[]>) => string,
  valueFn: (e: T, i: number, a: Readonly<T[]>) => R = IDENTICAL
) {
  const map: { [key: string]: R } = {}
  list.forEach((e, i, a) => {
    map[keyFn(e, i, a)] = valueFn(e, i, a)
  })
  return map
}

export function val<T>(val: T | undefined, defVal: T) {
  if (val === undefined) return defVal
  return val
}

export function isRootPath(path: string) {
  return path === ''
}

export function isAdmin(user?: User) {
  return !!(
    user &&
    user.groups &&
    user.groups.findIndex((g) => g.name === 'admin') !== -1
  )
}

export function wait(ms: number) {
  return new Promise<void>((resolve) => {
    setTimeout(resolve, ms)
  })
}

export const TASK_CANCELLED = { message: 'task canceled' }

export async function taskDone<T = any>(
  task_: PromiseValue<Task<T>>,
  cb?: Fn1<Task<T>, PromiseValue<false | undefined | void>>,
  interval = 1000
) {
  let task = await task_
  while (task.status === 'pending' || task.status === 'running') {
    if (cb && (await cb(task)) === false) throw TASK_CANCELLED
    try {
      task = await getTask(task.id)
    } catch (e: any) {
      if (e.status === 404) {
        throw TASK_CANCELLED
      }
      throw e
    }
    await wait(interval)
  }
  if (task.status === 'done') {
    cb && (await cb(task))
    return task.result
  } else if (task.status === 'error') {
    throw task.error
  } else if (task.status === 'canceled') {
    throw TASK_CANCELLED
  } else {
    console.warn('unknown task status', task)
    throw new Error('unknown error')
  }
}

const directives: O<Directive> = {
  markdown,
  longPress,
  focus,
  lazySrc,
}

export default {
  install(app) {
    Object.keys(directives).forEach((key) => {
      app.directive(key, directives[key])
    })
  },
} as Plugin

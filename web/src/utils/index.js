import { getTask } from '@/api'

import focus from './directives/focus'
import longPress from './directives/long-press'
import markdown from './directives/markdown'
import lazySrc from './directives/lazy-src'

export const IS_DEBUG = process.env.NODE_ENV === 'development'

export function setTitle(title) {
  if (title) title += ' - ' + window.___config___.appName
  else title = window.___config___.appName
  document.title = title
}

export function formatTime(d) {
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
export function formatBytes(bytes, decimals = 2) {
  if (bytes < 0) return '-'
  if (bytes === 0) return '0 B'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'K', 'M', 'G', 'T']

  let i = Math.floor(Math.log(bytes) / Math.log(k))
  if (i >= sizes.length) i = sizes.length - 1

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

export function formatPercent(n, based) {
  if (typeof n !== 'number') return ''
  if (typeof based === 'number') {
    if (based === 0) return ''
    n /= based
  }
  return (n * 100).toFixed(1) + '%'
}

export function dir(path) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return ''
  return path.substr(0, i)
}

export function filename(path) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return path
  return path.substr(i + 1)
}

export function filenameExt(filename) {
  if (!filename) return ''
  const i = filename.lastIndexOf('.')
  if (i === -1) return ''
  return filename.substr(i + 1).toLowerCase()
}

export function pathJoin(...segments) {
  return segments.filter(Boolean).join('/').replace(/\/+/g, '/')
}

export function pathClean(path) {
  if (!path) return ''
  const segments = path.split('/').filter(Boolean)
  const paths = []
  segments.forEach((s) => {
    if (s === '.') return
    if (s === '..') paths.pop()
    else paths.push(s)
  })
  return paths.join('/')
}

/**
 * remove element from array
 * @param {Array} array
 * @param {Function} e
 */
export function arrayRemove(array, e) {
  const index = array.findIndex(e)
  let el
  if (index >= 0) {
    el = array[index]
    array.splice(index, 1)
  }
  return el
}

export const debounce = (func, wait) => {
  let timeout
  return function executedFunction() {
    const later = () => {
      timeout = null
      func.apply(this, arguments)
    }
    clearTimeout(timeout)
    timeout = setTimeout(later, wait)
  }
}

// Returns a function, that, when invoked, will only be triggered at most once
// during a given window of time. Normally, the throttled function will run
// as much as it can, without ever going more than once per `wait` duration;
// but if you'd like to disable the execution on the leading edge, pass
// `{leading: false}`. To disable execution on the trailing edge, ditto.
export function throttle(func, wait, options) {
  let context, args, result
  let timeout = null
  let previous = 0
  if (!options) options = {}
  const later = function () {
    previous = options.leading === false ? 0 : Date.now()
    timeout = null
    result = func.apply(context, args)
    if (!timeout) context = args = null
  }
  return function () {
    const now = Date.now()
    if (!previous && options.leading === false) previous = now
    const remaining = wait - (now - previous)
    context = this
    args = arguments
    if (remaining <= 0 || remaining > wait) {
      if (timeout) {
        clearTimeout(timeout)
        timeout = null
      }
      previous = now
      result = func.apply(context, args)
      if (!timeout) context = args = null
    } else if (!timeout && options.trailing !== false) {
      timeout = setTimeout(later, remaining)
    }
    return result
  }
}

export function waitPromise(fn) {
  const promises = []
  let waiting = false
  return function () {
    if (!waiting) {
      waiting = true
      fn.apply(this, arguments)
        .then(
          (v) => {
            promises.forEach((p) => {
              p.resolve(v)
            })
          },
          (e) => {
            promises.forEach((p) => {
              p.reject(e)
            })
          }
        )
        .then(() => {
          promises.splice(0)
          waiting = false
        })
    }
    return new Promise((resolve, reject) => {
      promises.push({ resolve, reject })
    })
  }
}

const DEFAULT_VALUE_FN = (e) => e
export function mapOf(list, keyFn, valueFn = DEFAULT_VALUE_FN) {
  const map = {}
  list.forEach((e) => {
    map[keyFn(e)] = valueFn(e)
  })
  return map
}

export function val(val, defVal) {
  if (val === undefined) return defVal
  return val
}

export function isRootPath(path) {
  return path === ''
}

export function isAdmin(user) {
  return !!(
    user &&
    user.groups &&
    user.groups.findIndex((g) => g.name === 'admin') !== -1
  )
}

export function wait(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

export const TASK_CANCELLED = { message: 'task canceled' }

export async function taskDone(task, cb, interval = 1000) {
  task = await task
  while (task.status === 'pending' || task.status === 'running') {
    if (cb && (await cb(task)) === false) throw TASK_CANCELLED
    try {
      task = await getTask(task.id)
    } catch (e) {
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

const directives = {
  markdown,
  longPress,
  focus,
  lazySrc,
}

export default {
  /**
   * @param {import('vue').App} app
   */
  install(app) {
    Object.keys(directives).forEach((key) => {
      app.directive(key, directives[key])
    })
  },
}

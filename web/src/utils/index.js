
import { fileUrl, getTask } from '@/api'
import dayjs from 'dayjs'
import focus from './directives/focus'
import longPress from './directives/long-press'
import markdown from './directives/markdown'
import UiUtils from './ui-utils'

export const IS_DEBUG = process.env.NODE_ENV === 'development'

export function formatTime (d, toFormat) {
  const date = dayjs(d)
  if (!date.isValid()) return ''
  return date.format(toFormat || 'YYYY-MM-DD HH:mm:ss')
}

// from https://stackoverflow.com/a/18650828/8749466
export function formatBytes (bytes, decimals = 2) {
  if (bytes < 0) return '-'
  if (bytes === 0) return '0 B'

  const k = 1024
  const dm = decimals < 0 ? 0 : decimals
  const sizes = ['B', 'K', 'M', 'G', 'T']

  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
}

export function formatPercent (n, based) {
  if (typeof (n) !== 'number') return ''
  if (typeof (based) === 'number') {
    if (based === 0) return ''
    n /= based
  }
  return (n * 100).toFixed(1) + '%'
}

export function dir (path) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return ''
  return path.substr(0, i)
}

export function filename (path) {
  if (!path) return ''
  const i = path.lastIndexOf('/')
  if (i === -1) return path
  return path.substr(i + 1)
}

export function filenameExt (filename) {
  if (!filename) return ''
  const i = filename.lastIndexOf('.')
  if (i === -1) return ''
  return filename.substr(i + 1).toLowerCase()
}

export function pathJoin (...segments) {
  return segments.filter(Boolean).join('/').replace(/\/+/g, '/')
}

export function pathClean (path) {
  if (!path) return ''
  const segments = path.split('/').filter(Boolean)
  const paths = []
  segments.forEach(s => {
    if (s === '.') return
    if (s === '..') paths.pop()
    else paths.push(s)
  })
  return (path.charAt(0) === '/' ? '/' : '') +
    paths.join('/') +
    (path.charAt(path.length - 1) === '/' ? '/' : '')
}

/**
 * remove element from array
 * @param {Array} array
 * @param {Function} e
 */
export function arrayRemove (array, e) {
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
  return function executedFunction () {
    const later = () => {
      timeout = null
      func.call(this, arguments)
    }
    clearTimeout(timeout)
    timeout = setTimeout(later, wait)
  }
}

export function waitPromise (fn) {
  const promises = []
  let waiting = false
  return function () {
    if (!waiting) {
      waiting = true
      fn.apply(this, arguments).then(v => {
        promises.forEach(p => { p.resolve(v) })
      }, e => {
        promises.forEach(p => { p.reject(e) })
      }).then(() => {
        promises.splice(0)
        waiting = false
      })
    }
    return new Promise((resolve, reject) => {
      promises.push({ resolve, reject })
    })
  }
}

const DEFAULT_VALUE_FN = e => e
export function mapOf (list, keyFn, valueFn = DEFAULT_VALUE_FN) {
  const map = {}
  list.forEach(e => {
    map[keyFn(e)] = valueFn(e)
  })
  return map
}

export function val (val, defVal) {
  if (val === undefined) return defVal
  return val
}

export function isRootPath (path) {
  return path === ''
}

export function isAdmin (user) {
  return !!(user && user.groups && user.groups.findIndex(g => g.name === 'admin') !== -1)
}

export function wait (ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms)
  })
}

export async function taskDone (task, cb) {
  while (task.status === 'pending' || task.status === 'running') {
    await cb(task)
    task = await getTask(task.id)
  }
  return task
}

const filters = {
  formatTime, formatBytes
}

const directives = {
  markdown, longPress, focus
}

const utils = {
  formatTime, formatBytes, formatPercent,
  filenameExt, pathJoin, fileUrl, filename, dir
}

export default {
  install (Vue) {
    Vue.prototype.$ = utils
    Object.keys(filters).forEach(key => {
      Vue.filter(key, filters[key])
    })
    Object.keys(directives).forEach(key => {
      Vue.directive(key, directives[key])
    })

    Vue.use(UiUtils)
  }
}

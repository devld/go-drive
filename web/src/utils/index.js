
import dayjs from 'dayjs'
import markdown from './directives/markdown'

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
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']

  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(dm)) + ' ' + sizes[i]
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
  return segments.join('/').replace(/\/+/g, '/')
}

export function pathClean (path) {
  if (!path || path.charAt(0) !== '/') throw new Error('invalid path')
  const segments = path.split('/').filter(s => !!s)
  const paths = []
  segments.forEach(s => {
    if (s === '.') return
    if (s === '..') paths.pop()
    else paths.push(s)
  })
  return '/' + paths.join('/')
}

const DEFAULT_VALUE_FN = e => e
export function mapOf (list, keyFn, valueFn = DEFAULT_VALUE_FN) {
  const map = {}
  list.forEach(e => {
    map[keyFn(e)] = valueFn(e)
  })
  return map
}

const filters = {
  formatTime, formatBytes
}

const directives = {
  markdown
}

const utils = {
  formatTime, formatBytes, filenameExt, pathJoin
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
  }
}

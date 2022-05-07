import { buildURL } from '@/utils'
import axios, { API_PATH, axiosWrapper } from './axios'

const ACCESS_KEY = '_k'
const PROXY_KEY = 'proxy'

/**
 * @typedef {boolean|'cors'|'referrer'} UseProxy
 *
 * @typedef EntryMeta
 * @property {string} accessKey
 * @property {UseProxy} [useProxy]
 *
 * @typedef FileURLParams
 * @property {boolean} [noCache]
 * @property {UseProxy} [useProxy]
 */

export function listEntries(path) {
  return axiosWrapper.get(`/entries/${path}`)
}

export function getEntry(path) {
  return axiosWrapper.get(`/entry/${path}`)
}

export function searchEntries(path, q, next) {
  return axiosWrapper.get(`/search/${path}`, {
    params: { q, next },
  })
}

/**
 * @param {string} path
 * @param {EntryMeta} meta
 * @param {FileURLParams} params
 */
function _fileUrl(path, meta, params) {
  const query = {}
  if (meta?.accessKey) {
    query[ACCESS_KEY] = meta.accessKey
  }
  if (meta?.useProxy || params?.useProxy) {
    const useProxy = meta.useProxy
    const getType = params.useProxy
    let proxy = false
    if (getType === true) proxy = true
    else if (getType) {
      // if drive's backend says it checks referrer.
      // because embedded images or XHR always send Referer.
      // so we need proxy.
      // or drives's backend says it only disallow CORS,
      // but do not check referrer.
      // so we only use proxy when sending XHR request.
      proxy =
        useProxy === 'referrer' || (useProxy === 'cors' && getType === 'cors')
    }
    if (proxy) {
      query[PROXY_KEY] = '1'
    }
  }
  if (params?.noCache) query.r = Math.random()
  return buildURL(`/content/${path}`, query)
}

/**
 * @param {string} path
 * @param {EntryMeta} meta
 * @param {FileURLParams} params
 */
export function fileUrl(path, meta, params) {
  return `${API_PATH}${_fileUrl(path, meta, params)}`
}

/**
 * @param {string} path
 * @param {EntryMeta} meta
 */
export function fileThumbnail(path, meta) {
  const query = {}
  if (meta?.accessKey) {
    query[ACCESS_KEY] = meta.accessKey
  }
  return buildURL(`${API_PATH}/thumbnail/${path}`, query)
}

/**
 * @param {string} path
 * @param {EntryMeta} meta
 * @param {FileURLParams} params
 */
export function getContent(path, meta, params) {
  return axiosWrapper.get(
    _fileUrl(path, meta, {
      ...params,
      useProxy: 'cors',
    }),
    {
      transformResponse: [],
      _noAuth: true,
    }
  )
}

export function makeDir(path) {
  return axios.post(`/mkdir/${path}`)
}

export function deleteEntry(path) {
  return axios.delete(`/entry/${path}`)
}

export function copyEntry(from, to, override) {
  return axios.post('/copy', null, {
    params: { from, to, override: override ? '1' : '' },
  })
}

export function moveEntry(from, to, override) {
  return axios.post('/move', null, {
    params: { from, to, override: override ? '1' : '' },
  })
}

export function getTasks(group) {
  return axiosWrapper.get('/tasks', {
    params: { group },
  })
}

export function getTask(id) {
  return axiosWrapper.get(`/task/${id}`)
}

export function deleteTask(id) {
  return axios.delete(`/task/${id}`)
}

/// auth

export function login(username, password) {
  return axios.post('/auth/login', {
    username,
    password,
  })
}

export function logout() {
  return axios.post('/auth/logout')
}

export function getUser() {
  return axiosWrapper.get('/auth/user')
}

export function getConfig() {
  return axiosWrapper.get('/config')
}

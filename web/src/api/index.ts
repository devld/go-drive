import {
  Config,
  Entry,
  EntryMeta,
  EntryMetaUseProxy,
  SearchResult,
  Task,
  User,
} from '@/types'
import { buildURL } from '@/utils'
import http, { API_PATH } from './http'
import defaultHttp from '@/utils/http'

const ACCESS_KEY = '_k'
const PROXY_KEY = 'proxy'

export interface FileURLParams {
  noCache?: boolean
  useProxy?: EntryMetaUseProxy
}

export function listEntries(path: string) {
  return http.get<Entry[]>(`/entries/${path}`)
}

export function getEntry(path: string) {
  return http.get<Entry>(`/entry/${path}`)
}

export function searchEntries(path: string, q: string, next?: number) {
  return http.get<SearchResult>(`/search/${path}`, {
    params: { q, next },
  })
}

function _fileUrl(path: string, meta: EntryMeta, params?: FileURLParams) {
  const query = {} as O<any>
  if (meta?.accessKey) {
    query[ACCESS_KEY] = meta.accessKey
  }
  if (meta?.useProxy || params?.useProxy) {
    const useProxy = meta.useProxy
    const getType = params?.useProxy
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
  return buildURL(`/content/${path}`, query)!
}

export function zipUrl() {
  return `${API_PATH}/zip`
}

export function fileUrl(path: string, meta: EntryMeta, params?: FileURLParams) {
  return `${API_PATH}${_fileUrl(path, meta, params)}`
}

export function fileThumbnail(path: string, meta: EntryMeta) {
  const query = {} as O<any>
  if (meta?.accessKey) {
    query[ACCESS_KEY] = meta.accessKey
  }
  return buildURL(`${API_PATH}/thumbnail/${path}`, query)!
}

export function getContent(
  path: string,
  meta: EntryMeta,
  params?: FileURLParams
) {
  return defaultHttp
    .get(
      fileUrl(path, meta, {
        ...params,
        useProxy: 'cors',
      })!,
      { transformResponse: [] }
    )
    .then((res) => res.data)
}

export function makeDir(path: string) {
  return http.post<Entry>(`/mkdir/${path}`)
}

export function deleteEntry(path: string) {
  return http.delete<Task<void>>(`/entry/${path}`)
}

export function copyEntry(from: string, to: string, override?: boolean) {
  return http.post<Task<Entry>>('/copy', null, {
    params: { from, to, override: override ? '1' : '' },
  })
}

export function moveEntry(from: string, to: string, override?: boolean) {
  return http.post<Task<Entry>>('/move', null, {
    params: { from, to, override: override ? '1' : '' },
  })
}

export function getTasks<T>(group: string) {
  return http.get<Task<T>[]>('/tasks', {
    params: { group },
  })
}

export function getTask<T>(id: string) {
  return http.get<Task<T>>(`/task/${id}`)
}

export function deleteTask(id: string) {
  return http.delete<void>(`/task/${id}`)
}

/// auth

export function login(username: string, password: string) {
  return http.post<void>('/auth/login', {
    username,
    password,
  })
}

export function logout() {
  return http.post<void>('/auth/logout')
}

export function getUser() {
  return http.get<User | undefined>('/auth/user')
}

export function getConfig(optKeys: string[]) {
  return http.get<Config>('/config', {
    params: {
      opts: optKeys.join(','),
    },
  })
}

import {
  RawConfig,
  Entry,
  EntryMeta,
  EntryMetaUseProxy,
  SearchResult,
  Task,
  User,
} from '@/types'
import { buildURL } from '@/utils'
import type { HttpResponse } from '@/utils/http'
import http, {
  ACCESS_KEY,
  API_PATH,
  RETURN_RESPONSE_CONTEXT_KEY,
  binaryHttp,
  clearCachedPathPasswords,
  clearToken,
  pathPasswordHeaders,
  setToken,
} from './http'

const PROXY_KEY = 'proxy'

export interface FileURLParams {
  noCache?: boolean
  useProxy?: EntryMetaUseProxy
}

export function listEntries(path: string) {
  return http.get<Entry[]>('/list', {
    headers: pathPasswordHeaders(path),
    params: { path },
  })
}

export function getEntry(path: string) {
  return http.get<Entry>('/stat', {
    headers: pathPasswordHeaders(path),
    params: { path },
  })
}

export function searchEntries(path: string, q: string, next?: number) {
  return http.get<SearchResult>('/search', {
    params: { path, q, next },
  })
}

function _fileUrl(path: string, meta: EntryMeta, params?: FileURLParams) {
  const query = { path } as O<any>
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
  return buildURL('/download', query)!
}

export function fileUrl(path: string, meta: EntryMeta, params?: FileURLParams) {
  return `${API_PATH}${_fileUrl(path, meta, params)}`
}

export function getBlobContent(
  path: string,
  meta: EntryMeta,
  params?: FileURLParams
) {
  return binaryHttp
    .get<Blob>(
      fileUrl(path, meta, {
        ...params,
        useProxy: 'cors',
      }),
      { headers: pathPasswordHeaders(path) }
    )
    .then((blob) => blob)
}

export function getContent(
  path: string,
  meta: EntryMeta,
  params?: FileURLParams
) {
  return getBlobContent(path, meta, params).then((blob) => blob.text())
}

export interface ArtifactInfo {
  name?: string
  mimeType?: string
  size: number
  ref: string
}

export type ArtifactPrepareResult =
  | { info: ArtifactInfo }
  | { task: Task<ArtifactInfo> }

function artifactQuery(
  path: string,
  meta: EntryMeta,
  extra?: { args?: string; ref?: string }
) {
  const query = { path } as O<any>
  if (extra?.args) query.args = extra.args
  if (extra?.ref) query.ref = extra.ref
  if (meta?.accessKey) query[ACCESS_KEY] = meta.accessKey
  return query
}

export function artifactURL(
  path: string,
  meta: EntryMeta,
  handler: string,
  extra?: { args?: string; ref?: string }
) {
  return buildURL(
    `${API_PATH}/artifact/${handler}`,
    artifactQuery(path, meta, extra)
  )!
}

function preparedArtifact(
  response: HttpResponse<ArtifactInfo | Task<ArtifactInfo>>
): ArtifactPrepareResult {
  if (response.status === 202) {
    return { task: response.data as Task<ArtifactInfo> }
  }
  return { info: response.data as ArtifactInfo }
}

export function prepareArtifact(
  path: string,
  meta: EntryMeta,
  handler: string,
  args = ''
) {
  return http
    .post<HttpResponse<ArtifactInfo | Task<ArtifactInfo>>>(
      `/artifact/${handler}`,
      args,
      {
        headers: {
          ...pathPasswordHeaders(path),
          'content-type': 'text/plain',
        },
        params: artifactQuery(path, meta),
        context: { [RETURN_RESPONSE_CONTEXT_KEY]: true },
      }
    )
    .then(preparedArtifact)
}

export function getArtifact<T>(
  path: string,
  meta: EntryMeta,
  handler: string,
  extra?: { args?: string; ref?: string }
) {
  return http.get<T>(`/artifact/${handler}`, {
    headers: pathPasswordHeaders(path),
    params: artifactQuery(path, meta, extra),
  })
}

export function makeDir(path: string) {
  return http.post<Entry>('/mkdir', null, { params: { path } })
}

export function deleteEntry(path: string) {
  return http.post<Task<void>>('/delete', null, { params: { path } })
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

export function getTask<T>(id: string) {
  return http.get<Task<T>>(`/tasks/${id}`)
}

export function deleteTask(id: string) {
  return http.delete<void>(`/tasks/${id}`)
}

/// auth

export interface LoginResult {
  token: string
  expiresAt: number
}

export function login(provider: string, formData: Record<string, string>) {
  return http
    .post<LoginResult>(
      `/auth/${encodeURIComponent(provider)}/callback`,
      formData
    )
    .then((res) => {
      setToken(res.token)
      return res
    })
}

export function logout() {
  return http.post<void>('/auth/logout').finally(() => {
    clearToken()
    clearCachedPathPasswords()
  })
}

export function getUser() {
  return http.get<User | undefined>('/auth/user')
}

export function getConfig(optKeys: string[]) {
  return http.get<RawConfig>('/config', {
    params: {
      opts: optKeys.join(','),
    },
  })
}

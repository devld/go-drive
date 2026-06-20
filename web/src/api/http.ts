import { getLang } from '@/i18n'
import {
  Http,
  HttpError,
  HttpRequestBaseConfig,
  HttpRequestConfig,
  HttpResponse,
} from '@/utils/http'
import { createHttp } from '@/utils/http/http'
import {
  transformErrorResponse,
  transformJSONRequest,
  transformJSONResponse,
} from '@/utils/http/transformers'

export const AUTH_PARAM = 'token'

const AUTH_HEADER = 'Authorization'
const PATH_PASSWORD_HEADER = 'X-Path-Password'
const TOKEN_KEY = 'token'
const RESPONSE_HEADER_KEY = 'x-response'

const MAX_RETRY = 1

let apiPath = window.___config___.api
if (!/^https?:\/\//.test(apiPath)) {
  apiPath = location.origin + apiPath
}
export const API_PATH = apiPath

const BASE_CONFIG: HttpRequestBaseConfig = {
  baseURL: API_PATH,
  timeout: 60000,
}

export function setToken(token: string) {
  return localStorage.setItem(TOKEN_KEY, token)
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

// Cache of path passwords keyed by the path the user unlocked, persisted in
// localStorage. The password applies to the whole subtree, so it is attached
// automatically when requesting that path or any of its descendants.
const PATH_PASSWORDS_KEY = 'pathPasswords'

type CachedPathPassword = { path: string; password: string }

function loadCachedPathPasswords(): CachedPathPassword[] {
  try {
    const raw = localStorage.getItem(PATH_PASSWORDS_KEY)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

let pathPasswords: CachedPathPassword[] = loadCachedPathPasswords()

function saveCachedPathPasswords() {
  localStorage.setItem(PATH_PASSWORDS_KEY, JSON.stringify(pathPasswords))
}

export function setCachedPathPassword(path: string, password: string) {
  const existing = pathPasswords.find((p) => p.path === path)
  if (existing) existing.password = password
  else pathPasswords.push({ path, password })
  saveCachedPathPasswords()
}

export function clearCachedPathPasswords() {
  pathPasswords = []
  localStorage.removeItem(PATH_PASSWORDS_KEY)
}

function getCachedPathPassword(path: string): string | undefined {
  let best: { path: string; password: string } | undefined
  for (const p of pathPasswords) {
    const match = path === p.path || path.startsWith(p.path ? p.path + '/' : '')
    if (match && (!best || p.path.length > best.path.length)) best = p
  }
  return best?.password
}

export function pathPasswordHeaders(path: string): Record<string, string> {
  const pw = getCachedPathPassword(path)
  return pw ? { [PATH_PASSWORD_HEADER]: pw } : {}
}

async function processConfig(config: HttpRequestConfig) {
  if (!config.context) config.context = {}
  if (config.context._t === undefined) config.context._t = -1
  config.context._t++
  if (config.context._t > MAX_RETRY)
    throw new HttpError(-1, 'max retry reached')

  if (!config.headers) config.headers = {}

  const token = getToken()
  if (token) {
    config.headers[AUTH_HEADER] = token
    config.context._tokenUsing = token
  } else {
    delete config.headers[AUTH_HEADER]
  }

  config.headers['Accept-Language'] = getLang()

  return config
}

async function handlerError(e: any) {
  const response = e.response as HttpResponse
  if (!response) throw e
  const status = response.status

  const config = response.request
  if (status === 401) {
    // the token is invalid/expired: drop it and retry once as anonymous
    if (getToken() && getToken() === config.context?._tokenUsing) {
      clearToken()
    } else {
      // already anonymous (or token already changed); nothing to recover
      throw e
    }
    const originalHttp = config.context?.__initiator as Http | undefined
    if (!originalHttp) throw e
    return originalHttp(config)
  }

  throw e
}

export interface StreamHttpResponse<T = any> {
  status: number
  data: T
  stream: ReadableStream
}

export const streamHttp = createHttp<StreamHttpResponse>({
  ...BASE_CONFIG,
  transformRequest: [
    processConfig,
    transformJSONRequest,
    (config) => ({ ...config, onUploadProgress: undefined }),
  ],
  transformResponse: [
    async (error, resp): Promise<StreamHttpResponse> => {
      if (error) throw error
      let responseData = resp.headers[RESPONSE_HEADER_KEY]
      if (responseData) responseData = JSON.parse(responseData)
      return {
        status: resp.status,
        data: responseData,
        stream: resp.data,
      }
    },
  ],
})

export default createHttp({
  ...BASE_CONFIG,
  transformRequest: [processConfig, transformJSONRequest],
  transformResponse: [
    transformJSONResponse,
    transformErrorResponse,
    (error, resp) => {
      if (error) return handlerError(error)
      return resp.data
    },
  ],
})
